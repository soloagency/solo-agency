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
different accounts) — `error` says which.

**The cost of this pass is the cost of LOADING, not of thinking.** Measured 2026-08-21: a full
enrich envelope is ~5.7 KB (largest 17 KB), so a batch of 25 was ~35k tokens before the model
reasoned about anything, and the 860-record backlog would have been ~1.2M tokens of input alone.
Most of that weight is collector telemetry no verdict can use — `graphql_by_surface`,
`discovered_tabs`, `checked`, `elapsed_ms` — plus `about` repeating `about_lines`.

`pending` now returns a COMPACT record by default: identity, category, work, location, websites,
contact fields, the intro/about prose, and at most three clamped post captions. On the live data
that is **30% of the bytes** — ~56k tokens down to ~16.5k for 37 records — with the telemetry gone
and nothing a verdict needs removed.

### The pass is TWO passes, and only the first one reads paragraphs

Facebook does not state an industry, and there is no regex for it: on the live set `category` is
present on 22% of records and `work` on 40%, while the prose that states the trade is present on
94%. A model has to read it. **Trimming that prose to save tokens is the wrong lever** — a first
version clamped it to 600 characters, which truncated 33 of 36 live profiles and would have had
the decider guessing at the one field the verdict turns on.

The right lever is to read it **once, with the cheapest model that can read**, and never again.

**Pass 1 — extract (lowest-tier model, no judgement).** This is not a decision; it is turning
unstructured text into one value from a closed list. Give the sub-agent the full records and the
dictionary, and take back JSONL:

```
tool crm-store --client-dir {outreach} harvest pending --campaign X --limit 25 --unclassified --full
```

Each line back: `{"profile_url": "…", "industry": "<verbatim from lead_industries.json>",
"role": "…", "location": "…", "signals": ["the phrase it was read from"], "confidence": 0.0-1.0,
"classified_by": "<model>"}`. If it genuinely cannot tell, `{"unclear": true}` — an honest blank
beats an invented industry, and the code refuses a classification that is neither.

```
tool crm-store --client-dir {outreach} harvest classify --campaign X --file industries.jsonl
```

The industry is validated against **the same 43-entry dictionary the CRM gate enforces**
(`lead_industries.json`), verbatim, case and punctuation included. "real estate" is not
"Real Estate". A value that would be silently dropped at `contact add` is refused here instead,
where you can still see why — and the command echoes the allowed list on failure.

Rules that make this cheap, and it stops being cheap the moment any of them is broken:

- **Lowest tier, and no silent upgrade.** Extraction is not reasoning. If the thread starts
  growing, stop at a checkpoint file and open a fresh short one rather than dragging context.
- **File in, file out.** The extractor's output must never travel through a chat transcript —
  that is how a sub-agent's output becomes the supervisor's input and gets paid for twice.
- **Never re-classify.** `--unclassified` exists so the second run reads only what nobody has
  read yet.

**The industry travels on its own.** A `kept` verdict with a `lead_id` now carries the classified
industry onto the contact itself, through the same `enrich write` path the CRM gate guards. You do
not pass it, and you cannot forget it: the system read that field in pass one, and a contact
shipped without it was the system throwing away something it already knew. If the value is
refused, the verdict still stands and `industry_notes` says which one and why — a decision already
recorded is never undone by a field that did not land.

**Pass 2 — judge.** `pending` now serves the classification and DROPS the paragraphs it came from,
because the fact is already known. The verdict is then goal versus a structured record, which any
model can do at a fraction of the cost. The prose is still on disk: `--full` re-reads it for the
record that is genuinely ambiguous, which is a choice rather than a tax on all 25.

**Why the prose is NOT trimmed hard when a record is still unclassified.** The verdict turns on the lead's
INDUSTRY, and the structured fields that would state it are mostly absent: on the live set
`category` is present on 22% of records and `work` on 40%, while intro/about prose is present on
94%. The industry lives in the prose. A first version clamped it to 600 characters, which looked
like an 18% record — and truncated 33 of 36 live profiles, saving tokens by discarding the one
field the decision is made from. The budget is now 4000 characters: twelve points more expensive,
and it cuts nothing. **Never trade the deciding field for a cheaper batch.**

If a record still leaves the industry genuinely unclear, `--full` on THAT record is the correct
move; guessing is not.

Three rules follow, and they are the difference between a cheap pass and an expensive one:

- **Judge from the compact record.** `--full` exists for the record that is genuinely ambiguous
  after you have read the compact one. Paying for the full envelope on all 25 to serve the one is
  exactly what made this expensive.
- **Reject first.** A verdict that needs no contact write costs nothing but the read. Do the
  website/email ladder only for `kept`.
- **Apply in ONE call.** `harvest decide-batch --file <jsonl>` (or `--json '{"decisions":[…]}'`)
  takes every verdict in the batch at once. One bad verdict is reported and skipped; the other
  twenty-four still land. A turn per profile costs more in overhead than the verdict itself now
  costs in tokens.

```
tool crm-store --client-dir {outreach} harvest decide-batch --campaign X --file verdicts.jsonl
```

Each line: `{"profile_url": "…", "status": "kept|rejected|enrich_failed", "lead_id": "…", "reason": "…"}`.
Writing the verdicts to a FILE also keeps them out of the chat transcript — the sub-agent's output
never becomes the supervisor's input.

**You are the SUPERVISOR of this pass, not the judge. Delegation is mandatory, not an
optimisation.** Never read envelopes yourself: spawn low-level judge sub-agents (cheapest model
that can follow the rule), hand each ONE batch, and let it both decide and record. Then keep
driving until the queue is empty.

Why it is mandatory, stated plainly because the reminder alone has already failed: one envelope
carries a whole enrich record — `about_lines`, `work[]`, and every post caption. Twenty-five of
them is a large read, and after two or three batches the supervising context is full, so the run
"wraps up" while thousands still wait. That is not hypothetical: on 2026-08-18 a live campaign
held **860 profiles awaiting decision** against 99 ever decided, while the daemon kept adding
more every day. Delegating keeps those records out of your context entirely — you see one line
back per batch, so batch 40 costs you exactly what batch 1 did.

Each judge sub-agent gets: the campaign `goal_description` + `goal_keywords`, its own batch of
envelopes, and the rules below. It decides, calls `contact add` / `harvest decide` **itself**,
and returns ONE line: `n kept, n rejected, n failed, <uid of anything it could not decide>`.
It returns no prose, no envelope contents, and no reasoning — that is the whole point.

For EACH envelope, decide against the GOAL only:

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

### The supervisor loop — run it until it is actually empty

```
remaining = (harvest pending --campaign X --limit 1).remaining
while remaining > 0 and time/budget left:
    dispatch judge sub-agents over the next batches   (parallel where the runtime allows)
    re-read `remaining` from `harvest pending`        (never from your own count)
    log one line: "batch N: k kept, r rejected — M remaining"
```

Three rules that make the difference between finishing and appearing to:

- **`remaining` is read from the store, never inferred.** A sub-agent that dies silently leaves
  its batch undecided; only the store knows. Trust the number, not your tally.
- **Finishing one wave is not finishing the pass.** Do not report, do not summarise, do not move
  to the next step of the run while `remaining > 0` and there is time left. "I processed a
  batch" is not an outcome; "`remaining` is 0" is.
- **Stopping early is allowed, but only out loud.** If the run's time or budget ends first, say
  the exact number left — `stopped with 612 remaining` — in the run reply. A pass that quietly
  ends with a backlog reads identically to a pass that finished, which is how 860 accumulated
  without anyone noticing.

Every decision is one `decide` call; the daemon reads the registry, so an undecided record just
waits — nothing is lost between runs. But nothing moves either, and the daemon adds more
tomorrow.

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
- **Never judge in your own context, and never stop mid-queue.** This applies to a scheduled
  run and equally to the operator asking in chat ("duyệt list lead"): spawn low-level judge
  sub-agents, keep supervising, and stop only when `remaining` is 0 or you say out loud how many
  are left. An operator who asks for a list to be reviewed is asking for the LIST, not for a
  batch of it — a pass that ends after 25 of 860 and reports success has answered a different
  question than the one asked.
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
  budget, collectors live. The Campaigns page shows the same progress line. Two states get an
  extra sentence, because nothing else announces them: (1) `harvest status` shows every seed
  `exhausted` (or errored/removed) with an empty queue → "campaign X finished its walk: N kept,
  M rejected in total — add seed profiles or pause it" (the daemon does not pause it on its
  own; nothing is lost either way); (2) any seed carries an `error` → surface it verbatim.
- **Reaching the daily budget is not an event.** The daemon stops enqueuing for the day and
  resumes after midnight (local) outside quiet hours, from the same cursor and queue; the
  awaiting-decision records still wait for you. Do not report it as a blocker.

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
