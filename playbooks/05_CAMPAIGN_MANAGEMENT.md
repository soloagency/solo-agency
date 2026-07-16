# Stage 5 — Campaign Management

## Load Rule

Load this stage when creating or editing a campaign, defining a segment, or populating a
campaign's enrich queue — in Setup Flow (first campaign) or Automation Flow (new cold/trigger
campaign, or the daily "load new pipeline" step). Print a LOAD LEDGER per
`playbooks/LOAD_LEDGER_PROTOCOL.md` before acting.

## Hard Gates For This Stage

- A campaign's **goal is the writing blueprint, not a label**: `goal_type` drives the email
  structure Stage 6 produces and the `success_event` that Stage 10 / the rules engine fire.
- **One audience segment per campaign.** A segment is a saved flat-field filter; it is resolved
  through `crm_store.py`, which already excludes suppressed identities, merged tombstones, and
  `do_not_contact` — never hand-roll audience selection.
- **No guessing.** In the MVP nothing produces a guessed email address. An `email_first` campaign
  only queues contacts that already have a **found** email; a contact with no email is skipped
  (or handled via assisted channels later), never emailed at a guessed address.
- **Don't double-touch.** The enrich-queue populator skips a contact that is already queued/sent
  in this campaign, is mid-sequence (frozen after a reply) anywhere, or was emailed by **another**
  campaign within `min_days_between_touches_across_campaigns` (default 7).
- All of the above is enforced **in `crm_store.py`**, not in prose. Do not populate a queue or
  select an audience by reading/writing files directly.

## Source Preservation Rule

Campaign config (`campaigns/{slug}/campaign_config.json`) and segments (`crm/segments.json`) are
config, but every CONTACT read for audience resolution goes through `crm_store.py`. When any
instruction here disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.

## 1. The goal object (Stage 6 consumes it)

`goal_type` ∈ `book_meeting | get_reply | direct_sale | reactivation | nurture_upsell | event_invite`.
It selects the email structure in Stage 6 (e.g. `book_meeting` → short, one time-bound CTA;
`get_reply` → ends on a question, no link; `direct_sale` → value + one offer link). The full goal:

```json
{"goal_type": "book_meeting",
 "objective": "get the realtor to accept a free sample video for their newest listing",
 "offer": "1 free vertical video from their current listing photos",
 "value_proposition": "listings with video tours in AL metros sell 2-3x faster",
 "proof_points": [{"claim": "agent X in Hoover +40% inquiries", "evidence_url": "https://..."}],
 "cta": {"type": "reply_yes", "text": "Reply 'sample' and I'll send it over today"},
 "success_event": {"on": "reply_positive", "create_deal_stage": "new_reply"}}
```

`success_event` wires straight into the rules engine: a positive reply on this campaign creates a
deal at the named stage (Stage 10 / `crm_store.py apply-rules`).

## 2. Define a segment

A segment is a saved filter over flat contact fields (the Cond DSL, §6): `[field, op, value]`,
`op ∈ {=, !=, <, >, contains, in}`. Identity lookups are NOT expressible here (they use the
reverse index); a segment filters on `lifecycle_stage`, `tags`, `custom_fields.*`, `tz`, etc.

```sh
python3 tools/crm_store.py --client-dir DIR segment set --json \
  '{"id":"al-realtors-active","name":"Active AL realtors","where":[["lifecycle_stage","=","lead"],["custom_fields.state","=","AL"]]}'
python3 tools/crm_store.py --client-dir DIR segment resolve --id al-realtors-active   # preview the audience
```

`resolve` already drops suppressed / merged / `do_not_contact` contacts.

## 3. Create a campaign

```sh
python3 tools/crm_store.py --client-dir DIR campaign create --slug demo-outreach --json \
  '{"goal":{"goal_type":"book_meeting","objective":"book a demo","cta":{"type":"reply_yes","text":"Reply YES"}},
    "audience":{"segment":"al-realtors-active","personalization":{"min_confidence":0.7,"no_hook_fallback":"generic_honest_opener"}},
    "sendboxes":["sb-a"],"daily_quota":40,"channel_strategy":"email_first"}'
```

Defaults are filled in for any field you omit: a 4-step sequence (step 1 cold + 3 bumps with
`gap_days` 4/5/7, the last a breakup), `approval_mode: manual_all`,
`min_days_between_touches_across_campaigns: 7`, `guardrails.no_fake_re: true`. `campaign get` /
`campaign list` read them back. Creating a campaign makes its `queue/`, `outbox/pending_approval/`,
`outbox/approved/`, and `history/` folders.

## 4. Populate the enrich queue (the JIT buffer)

The daily run pulls a small buffer (3–7 days of volume) rather than enriching the whole pool
up front — hook freshness is the whole point. Populate it with:

```sh
python3 tools/crm_store.py --client-dir DIR campaign queue --slug demo-outreach --limit 100
```

Output reports `queued` plus a `skipped` breakdown (`already_in_campaign`,
`recently_touched_elsewhere`, `in_active_sequence`, `no_email`). Re-running is safe — an
already-queued or already-sent lead is never re-queued. The queued leads then flow into Stage 4
(verify + enrich) and Stage 6 (draft) during the daily run.

## Completion Gates

- The campaign has a valid `goal_type` and a resolvable `audience.segment`.
- The segment was previewed with `segment resolve` and looks right to the operator (surface it
  in an `**[ACTION REQUIRED]**` for confirmation on first setup).
- The enrich queue was populated via `campaign queue` (never by editing `enrich_queue.jsonl`).
- No campaign targets guessed addresses; `email_first` campaigns skip no-email contacts.

## Phase status

2A (this stage's tooling — campaign/segment/queue in `crm_store.py`) is **built**. The stages this
hands off to — Stage 4 verify/enrich (skill `email-verify-enrich`), Stage 6 email writing (skill
`email-writing`), Stage 10 follow-up — ship in Phase 2 milestones 2B–2D; where they are still
`status: planned`, follow DESIGN §22 R1 (load the covering DESIGN section, record
`stage_file_pending`), never a tool-missing blocker or GitHub re-fetch.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
