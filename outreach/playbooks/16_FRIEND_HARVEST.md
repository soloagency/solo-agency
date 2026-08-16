# Leads From Friends — the `friend_harvest` campaign channel

Stage: `16`

## Load Rule

Load whenever a client has a campaign with `channel_strategy: friend_harvest` — during
Setup Flow when one is created or edited, and in EVERY scheduled run's step for collector-
executed channels (AUTOMATION_SCHEDULING §H). LOAD LEDGER applies.

## What this channel is

The friend list of a well-connected professional is the highest-quality lead source this
system has: people who already trust the seed, in the seed's own trade and town. It is also
the most account-sensitive read this system performs — hundreds of profile opens a day —
so the WALK is done by the bridge daemon, in code, and the agent does only the judgement.

- **The daemon** (persistent bridge, `harvest_daemon.go`): reads each seed's friend list in
  legs (`fb.profile.friends` with `start_cursor`/`end_cursor`, ~10 pages a leg), drops friends
  already in the client-wide `harvest/seen_profiles.json`, enriches each new friend ONCE with
  `fb.profile.enrich` (20–40 s apart, random), rotates the collector account every job, honours
  the daily budget, the per-collector budget and the quiet hours (all in `/ui/settings`, per-
  campaign overrides in the campaign), and parks each enriched record in
  `harvest/{campaign}/enriched/` for the agent. It sends NOTHING. Nothing here needs approval.
- **The agent** (this stage): applies the campaign GOAL to each enriched record and records the
  verdict. A `kept` friend becomes a CRM contact immediately (the operator ruled: no approval
  gate — nothing leaves the system). A `rejected` one is remembered client-wide so no campaign
  ever enriches that person again.

## Setup (Setup Flow, or the operator in `/ui/{client}/campaigns`)

1. Create the campaign with `channel_strategy: friend_harvest`, a `goal.description` in the
   operator's words that names WHO to keep (industry / trade, location, any other trait —
   "realtors and loan officers in Orange County, Vietnamese-speaking a plus"), and
   `seed_profiles` (one profile url per line in the UI; each is normalized and de-duplicated,
   groups are refused — this walks a PERSON's friend list).
2. Optional `harvest` block: `goal_keywords` (lower-case words a friend-list subtitle may carry
   — "realtor", "loan", "insurance"; matches are enriched FIRST, non-matches are still enriched
   later, never dropped, because subtitles are often empty), `daily_budget` /
   `per_collector_budget` / `leg_pages` / `quiet_from` / `quiet_to` overrides. Blank = the
   operator's system settings (`/ui/settings`, "Leads From Friends" block).
3. Start it deliberately. A harvest campaign is CREATED PAUSED (it touches Facebook on its
   own, with no approval gate, so saving must never be the trigger): finish the seeds and the
   goal, save, then press **Start harvest** on the campaign page (or `campaign update
   --json '{"status":"active"}'`). Otherwise nothing else: the daemon reconciles `seed_profiles` with its progress on EVERY tick —
   a seed added later starts being walked, a seed removed stops (its cursor is kept, so
   re-adding resumes). `tool crm-store harvest seed --campaign X` is only a manual way to
   force that same sync. The daemon runs as long as the campaign `status` is `active` and at
   least one collector extension is checking in; `status: paused` stops the walk at its
   cursor, resuming continues from there.

## Every scheduled run — the judgement pass (§H, one step)

```
tool crm-store --client-dir {outreach} harvest pending --campaign X --limit 25
```

returns up to 25 pending envelopes plus the campaign's `goal_description` and `goal_keywords`.
Each envelope: `{uid, profile_url, seed, name, subtitle, ok, error?, attempts, record}` where
`record` is the `fb.profile.enrich` item (`name`, `category`, `about_lines[]`, `work[]`,
`emails[]`, `websites[]`, `posts[]` with caption/date/permalink, `checked[]`). `ok: false`
means the collector could not read the profile (private, error, or gave up after retries on
different accounts) — `error` says which. For EACH envelope, in a fresh-context judge
sub-agent when the batch is large (same isolation rule as email writing — a long loop degrades
judgement), decide against the GOAL only:

- **kept** — the friend matches the goal's trade AND location (and any other stated trait),
  evidenced by the record (`about_lines`, `work[]`, `posts[]` captions, subtitle). Then:
  (a) create-or-match the contact in ONE call — `tool crm-store --client-dir {outreach}
  contact add --json '{"name":{"full":"…","given":"…","entity_type":"person|company|page"},
  "identities":{"socials":{"facebook":"<profile_url>"},
  "emails":[{"address":"<email>","source":"friend_harvest","status":"unverified","is_primary":true}],
  "phones":[{"number":"<phone>","type":"cell","source":"friend_harvest"}]},
  "tags":["friend_harvest"],"custom_fields":{"source":"friend_harvest","harvest_seed":"<seed
  url>","harvest_campaign":"X"}}'` (emails/phones are LISTS OF OBJECTS — `{address}` /
  `{number}` — never plain strings: the store's identity index silently drops a bare string,
  which would create an un-mailable contact and defeat dedup; omit the key when the record has
  none) — it returns `lead_id` + `outcome` (`matched` when the
  identity already exists in the CRM: that contact is REUSED, never duplicated, per Stage 13;
  `created` otherwise); name/`given`/`entity_type` follow the enrich skill's addressing rules;
  (b) **finish the email ladder for a kept friend** — the enrich record already covered
  Facebook (bio + About sub-tabs, `checked[]`); if `emails[]` is empty but `websites[]` is not,
  run rows 6-7 of the Stage 4 ladder NOW, for this person only: fetch the website's
  Contact/Team/About page and footer (WebFetch, read-only), then the off-platform search
  when the site yields nothing. This is the cheapest moment to do it (only goal-matched
  people, one fetch each) and the reason the CRM ends up with a reachable lead instead of a
  name; a kept friend with a website and no email attempt is a Stage 4 violation. Record the
  outcome exactly as Stage 4 does: an address goes into `identity.channels_found.emails`
  (with its `evidence_url`), and `email_discovery` carries the enrich record's
  `{profile_url, emails, websites, found_on, checked}` plus the website surfaces you read,
  so `mark_email_not_found` is honest when nothing was published anywhere; then
  (c) `enrich write --contact <lead_id>` with the record's hooks (each with `evidence_url`)
  and the email findings, so the dossier is write-ready for any later campaign; (d) `tool
  crm-store --client-dir {outreach} harvest decide --campaign X --profile <profile_url
  exactly as the envelope's profile_url> --status kept --lead-id <lead_id> --reason "<one
  line: which goal trait matched, from which evidence>"`.
- **rejected** — does not match the goal (wrong trade, wrong place, no professional signal at
  all). `harvest decide --campaign X --profile <profile_url> --status rejected --reason "<one
  line>"`. Never create a contact "just in case": the client-wide seen registry guarantees this
  person is never enriched again, so a wrong reject is a lost lead — reject on evidence, not
  on absence of a subtitle.
- **enrich_failed** — the envelope is `ok: false` (private profile / unreadable after retries
  on different accounts). `harvest decide --campaign X --profile <profile_url> --status
  enrich_failed --reason "<the envelope's error>"`; it is remembered and skipped. (Transient
  failures never reach you: the daemon already retried them on other collectors.)

`decide` refuses a `--profile` that is not in the pending set — pass the envelope's
`profile_url` verbatim, never a re-typed variant.

Do not stop at one batch: loop `pending` → decide until `remaining` is 0 or the run's time is
up. Every decision is one `decide` call; the daemon reads the registry, so an undecided record
just waits — nothing is lost between runs.

## Rules that are not optional

- **Never enrich or walk anything yourself.** No `fb.profile.friends` / `fb.profile.enrich`
  jobs from the agent for a harvest campaign — the daemon owns pacing, rotation and budgets,
  and an agent job on top of it doubles the account footprint the whole design exists to
  bound. If the daemon looks stuck, report `harvest status` (queue, in_flight, day counters,
  last enrich time) in the run reply; do not "help".
- **Failover is the daemon's, not yours.** Collectors that stop checking in are simply not
  picked; a job that stalls is cancelled and its friend re-queued to ANOTHER account (max 3
  attempts, then it reaches you as `ok: false`); an account with 3 consecutive failures is
  quarantined for 2 hours and released by a probe; a seed whose friend list no account can
  read is flagged in `harvest status` (`seeds[].error`) — surface that one to the operator
  ("an account that is friends with this seed may be needed"), never re-run it yourself.
  Per-account daily caps and pacing are enforced operator-wide (`collector/harvest_ledger.json`)
  across every harvest campaign of every client.
- **The goal decides, not the agent's taste.** A friend who is clearly a great person but not
  the goal's trade/place is `rejected`. The operator widens the goal if they want more.
- **Only kept friends get the website hop.** Rows 6-7 cost a live fetch per person; spending
  them on friends the goal will reject is waste. Judge first, then hunt the address for the
  ones you keep — never the other way round.
- **Kept = in the CRM, not in a campaign.** Harvest fills the CRM. Whether a kept contact
  enters an email/DM campaign is a separate, normal decision (queue rules, approvals) — this
  channel never sends and never queues into another campaign on its own.
- **Report, plainly.** The run reply carries one line per harvest campaign: seed N of M,
  friends seen, queued, awaiting decision, kept / rejected this run, today's enrich count vs
  budget, collectors live. The Campaigns page shows the same progress line.

## Storage (Stage 7 addendum)

- `outreach/harvest/seen_profiles.json` — client-wide `{uid → status, seed, campaign, lead_id,
  reason, first_seen, updated_at}`; statuses `seed | queued | enriched | kept | rejected |
  enrich_failed`. Written by the daemon and by `harvest decide`; read by every ingest.
- `outreach/harvest/{campaign}/progress.json` — per seed `{url, uid, friends_url, end_cursor,
  legs_done, friends_seen, exhausted, last_leg_at, last_leg_box}`, `queue[]` (awaiting enrich,
  goal-keyword matches first), `in_flight{}`, `await_decision[]`, day counters, totals.
- `outreach/harvest/{campaign}/enriched/{uid_hash}.json` — the enrich record awaiting a verdict;
  deleted on `decide`.
- Campaign config: `channel_strategy: friend_harvest`, `seed_profiles[]` (clean store-form
  urls), `harvest{}` overrides.
- `{data root}/collector/harvest_ledger.json` — operator-wide per-collector ledger (day counts,
  last job time, consecutive failures, quarantine) — the single truth for caps, pacing and the
  circuit breaker across all harvest campaigns.
- Collector output for harvest jobs lands under the OWNING client's inbox tree
  (`collector/inbox/YYYY-MM/{owner}/harvest/{run_id}/`) even when another client's extension
  did the reading — a reader never keeps another client's lead material.
