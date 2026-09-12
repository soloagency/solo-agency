# Lead And Competitor Detection

Stage: `10`

## Load Rule

Load whenever the agent is about to detect, score, report, store, or improve lead and competitor opportunities.

Also load during:

- first agency run;
- scheduled daily run;
- private data source scan analysis;
- public research analysis when lead or competitor signals are possible;
- HTML report generation or repair;
- any human request about leads, competitors, comments, replies, opportunities, outreach, or competitor monitoring.

## Hard Gates For This Stage

- Lead and competitor detection is not a light report appendix. Treat it as a core agency function.
- Detect leads and competitors during the same public/private data collection pass used for ideas and market research. Do not add a separate extra scan unless the human explicitly asks.
- For the first lead and competitor pass for a client/source set, use 10 scrolls per approved private data source when Local Collector is active and safe.
- For recurring daily scheduled runs, use 5 scrolls per approved private data source by default, unless the saved Local Collector configuration is lower or the human explicitly configured a safer lower value.
- Source discovery mode is different: it may scroll deeper under Stage 8 rules. Do not confuse source discovery with daily lead/competitor monitoring.
- For private data sources, use only the Solo Agency Local Collector extension plus Local Collector app. Never use Claude in Chrome, Codex built-in browser, Playwright, Puppeteer, Selenium, or any agent-controlled browser for private data source collection.
- Every detected lead or competitor opportunity shown in the HTML report must include the post/current URL when available, context, classification, why it matters, and a copy-ready suggested comment.
- The HTML report keeps one `Lead & Competitor Opportunities` section, inside the one `Source Intelligence` section, grouped by source (Google/Web, Facebook, Instagram, X, industry sites, custom URLs — only the ones that ran), or the same meaning in the human/report language. Never split it into a public/private pair.
- Suggested comments are for human review only. Do not auto-comment, auto-DM, or initiate outreach without explicit human approval — the **send/act** side always needs a human (see the approve-then-send gate). **This stage never sends, even after approval**: its job ends at "here are qualified leads and a suggested comment for you to review." Approved agency-level outreach is executed in a different module (OutreachCRM Stages 16/17, the operator's own brand and accounts), never from this scan loop, and never as or on behalf of a client. Data **collection and analysis**, on the other hand, is consented by the operator's own setup + command: the agent may read, extract, and combine whatever the operator directs it to research — the operator's own business data (industry, goal, content) and the prospects/sources they point the collector at, including contact details (email/phone) — for lead-finding and email personalization. **The one absolute prohibition:** never read, store, or transmit the operator's own credentials or secrets (usernames, passwords, cookies, tokens, session/auth data, API keys). Do not bypass access controls or CAPTCHAs — read what the operator's own session already renders.
- Comments must build personal brand and trust by adding value. They must not directly advertise the user's service, attack competitors, or sound like generic AI text.

## Source Preservation Rule

This file is detailed source material. Do not summarize away definitions, scan-depth rules, reporting requirements, comment style requirements, storage fields, or completion gates.

---

## Two passes over a watched source

A scan of a private source answers two different questions and needs two different moves, both
bounded by `playbooks/skills/lead-engine/safety.md`:

- **Search** the source for the phrasings that mean somebody needs help right now (`tool
  source-keywords plan|urls` → `fb.group.search_posts`). This is where leads come from, because it
  asks for people with a need instead of reading whatever was posted most recently.
- **Scroll** the feed shallowly for everything else: the shift in what the community discusses, the
  competitor activity, and the phrasings worth adding to the bank.

The daily procedure is in `playbooks/04_DAILY_SCHEDULE.md` step 14. This stage still qualifies
whatever either pass returns — the passes acquire, this stage judges. When a hunt runs instead
(`playbooks/skills/lead-engine`, an interactive "find me leads" ask), it writes the terms that
worked back into the same bank, so tomorrow's monitoring inherits what the hunt discovered.

## Social Discovery Pass (step 11C of the daily run)

This pass is step 11C of `playbooks/04_DAILY_SCHEDULE.md` Daily Run Algorithm, inserted between 11B
and step 12/13 (the former step 11C, `notification_channel_missing`, is now 11D). The full spec
lives here; the scheduled-run call-out lives in `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` steps
12D/12E.

"Vòng khám phá xã hội" — the impressive first lead scan across Facebook, Instagram and X for a
client, and afterwards a light daily companion on all three. It runs INSIDE the client's existing
daily run task; there is no separate automation task for it. Setup Flow's step 4 ("Kết nối
Facebook, Instagram and X") installs the bridge and the client's extension — one unpacked extension
covers all three platforms — well before this pass ever runs (`playbooks/SETUP_FLOW_ENTRYPOINT.md`),
and this pass is normally the client's first-ever discovery pass on each platform — but "first-ever"
is tracked per platform by `facebook_discovery_first_pass_done` / `instagram_discovery_first_pass_done`
/ `x_discovery_first_pass_done` on the Client Intelligence Profile, not by which automation run
number it is: a client who postpones a platform at setup and connects it three runs later still gets
that platform's FIRST RUN budget below on that later run, because that is still the first time this
pass has ever executed for that platform.

Before this pass may run at all, at least one of `facebook_lead_source` / `instagram_lead_source` /
`x_lead_source` must be `enabled` (see `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 12D/12E and
`playbooks/SETUP_FLOW_ENTRYPOINT.md`). Each platform's own participation is then decided
independently by its own `{platform}_lead_source` field: `pending` (the human has not yet resolved
item 4 for that platform either way) skips that platform's steps quietly — no awareness line,
because nothing has been decided. If the human confirmed `web_only` for a platform (the
acknowledgment-based escape, never a bare "bỏ qua"/"để sau"), skip that platform's steps and carry
that platform in the persistent web-only awareness line (`playbooks/06_AGENCY_REPORT_STANDARD.md`)
instead, plus the one-line way to turn it back on. A platform whose first call in a run comes back
logged-out (extension login probe, or `stopped_because: logged_out`) is itself set to `web_only` for
the day with reason "not logged in on {date}", and named in the same awareness line — the human
re-enables it by logging in in Chrome; no setup step repeats.

### Re-probe on every run

A platform in `web_only` whose `{platform}_web_only_reason` starts with "not logged in" is
re-probed on every run: its step-1 call (`fb.search.posts` / `ig.search.posts` / `x.search.posts`
with the first discovery term) is issued in its normal round-robin slot as the probe. If it returns
data, the run sets `{platform}_lead_source` back to `enabled`, clears the reason, stamps
`{platform}_lead_source_updated_at`, and the platform continues the rotation normally — that probe
counts as its step 1. If it comes back logged-out again, the platform keeps `web_only`, the reason's
date is refreshed to today, it loses its remaining turns for the day, and the awareness line names
it. Cost: at most one call per such platform per run, inside its daily budget. A `web_only` chosen
by the human (any other reason) is never re-probed automatically — only the `social_web_only_upsell`
job re-enables it, with the FIRST RUN budget. When the extension later reports per-platform login
state at check-in (`playbooks/TODO.md` handoff), the check-in may flip the field earlier, but the
run-time probe stays the rule of record. Each probe, successful or not, stamps
`{platform}_last_login_probe_at` on the Client Intelligence Profile
(`playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`).

For the reader, three shapes this takes:

- One platform not logged in → it drops out of the rotation for the day, the other two run, the
  awareness line names it, next run re-probes.
- All three not logged in → the pass runs zero social calls this run, the run still reads Google +
  custom sources, the awareness line names all three, `social_login_reconnect` is offered, every
  later run re-probes with 3 calls total.
- Extension not connected at all → nothing changes from today's rule (`extension_health` stale →
  Login Reminder / reconnect job, platforms stay as they were).

### Platform table

| Platform | Ordered steps | FIRST RUN budget | DAILY budget |
|---|---|---|---|
| Facebook | 1 feed (`fb.search.posts`) → 2 people (`fb.people.search`) → 3 groups (`fb.groups.search`, readable groups only: public, or private where the account is a member) → 4 in-group (`fb.group.search_posts`, up to 20 monitored groups per run from the registry plan) | 9 discovery calls (3/3/3) + up to 20 monitored groups × 3 terms, `max_pages` ≤ 4 | 3 discovery calls (1/1/1) + up to 20 monitored groups × 2 terms |
| Instagram | 1 search (`ig.search.posts`) → 2 people (`ig.people.search`) → 3 profile depth (`ig.profile.posts` on the best hits) → 4 comments (`ig.post.comments` on the best posts) | ≤ 12 calls (3/3/3/3) | ≤ 4 calls (1/1/1/1) |
| X | 1 search Latest (`x.search.posts`) → 2 people (`x.people.search`) → 3 profile depth (`x.profile.posts` on the best hits) → 4 replies (`x.post.replies` on the best posts) | ≤ 12 calls (3/3/3/3) | ≤ 4 calls (1/1/1/1) |

Keyword source for all three platforms is the same discovery terms from `tool public-keywords ...
plan --kind discovery` (the `community_discovery` kind — see "Discovery terms" below). Instagram and
X people search plateau fast, so the DAILY tier rotates one new term per day per platform rather
than reusing the same one every day. FIRST RUN minimum 10 leads stays a floor across ALL PLATFORMS
COMBINED, not a stop, not a per-platform target (see "Lead target" below). Instagram and X have no
groups: nothing from either platform is added to the source registry as a group, and the registry's
Facebook group entries (`source_type: group`) stay Facebook-only.

Write actions exist on Instagram and X (`ig.post.react/comment/message.send`,
`x.post.like/reply/publish/dm.send` — `dm.send` parked) but are OUT OF SCOPE for this pass: it
reads, classifies, and scores; it never reacts, comments, replies, or messages. The outreach
playbooks adopt those write actions separately.

### Round-robin rule

Interleave platforms. Build the job list in rounds: round k = one Facebook job, then one Instagram
job, then one X job, in that order; each platform advances exactly one step per round. Never
enqueue two consecutive jobs on the same platform while another platform still has a pending step.
A platform that is not connected, not logged in, at its budget, or tripped simply loses its turn —
its slot is skipped, not given to another platform, so the spacing between two requests on the same
platform never shrinks. The minimum gap between two requests on the same platform is the Facebook
rule already in force (the collector queue runs one job at a time per client and the other two
platforms' jobs sit between). The safety trip is per platform: a checkpoint, rate-limit or
logged-out signal on one platform removes that platform from the rotation for the day and the
others continue.

### Qualify as you go

**Qualify as you go.** Every job's rows are qualified the moment that job's result comes back — never batched
to the end of the pass. The order inside one job is: read the result for a trip signal, then run the Lead
Qualification Rule over its rows, then `tool crm-store ... lead capture` for every hot/warm/watch row, then
`tool source-registry record --leads <n>` when the job was a group scan, then submit the next job. A run is
therefore producing CRM contacts from its FIRST collector call (a feed-search post can be a hot lead at minute
one), not at the end. Stage 5 `filter_leads` keeps its name but is now the reconciliation step: counts
re-read from the CRM (`contact lock-status`), Step 5 discovered-thread recording, and anything a per-job pass
left over — never the first time the rule is run.

### Progress: six stages

**Run Progress Rule (six stages, one line each).** Every run that executes the Social Discovery
Pass reports progress at six stage boundaries, mapped onto the round-robin rounds:

1. `find_posts` — round 1: `fb.search.posts` / `ig.search.posts` / `x.search.posts`;
2. `find_people` — round 2: people search on each platform;
3. `find_groups` — round 3: `fb.groups.search` (Facebook) / profile depth (Instagram, X);
4. `scan_in_group` — round 4: `fb.group.search_posts` (Facebook) / comments and replies (Instagram, X);
5. `filter_leads` — reconciliation, not the first qualification pass: counts re-read from the CRM
   (`contact lock-status`), Step 5 discovered-thread recording, and anything a per-job pass left
   over — qualification and CRM capture already ran per job from stage 1 onward ("Qualify as you
   go" above);
6. `build_report` — report_state, INTERNAL_REPORT, notification, standup line.

A stage is done when every platform still in rotation has finished its step for that round (a
tripped, skipped or budget-exhausted platform counts as done with its numbers as they stand). At each
boundary the run appends ONE JSON line to `daily-content-pipeline/automation/run_progress.jsonl`
(schema in `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`) with `tool run-progress --pipeline {setup-root}/daily-content-pipeline append --client
<slug> --stage find_people --calls-done 6 --calls-planned 45 ...` (one example of the shape; every
stage boundary uses the matching `--stage` value) before it submits the next round's jobs. When Sam
speaks about a run in progress (a background wake, or the Boss asking),
the update has exactly this shape, in the Boss's language, numbers read from the progress line just
read (Read-Before-Claim Rule), never from memory:

```text
Giai đoạn {k}/6 xong — {stage}: {the five result lines that apply so far, one number each}. Lead
đến giờ: {leads_hot} hot / {leads_warm} warm / {leads_watch} watch.
Đang chạy: {stage k+1}. Còn lại: {remaining stages}. Đã dùng {calls_done}/{calls_planned} lượt gọi.
Dự kiến xong: {HH:MM}–{HH:MM}.
```

Wait mechanics: `tools/wait_for_run <client_slug> <since_iso> --watch progress --timeout <s>` exits 0
and prints `progress <line>` on the first new `run_progress.jsonl` line for that client after
`since`, or `standup <line>` when the run's standup line lands first; exit 3 on timeout. Sam arms it
right after dispatch, reads `tool run-progress --pipeline {setup-root}/daily-content-pipeline show --client <slug>` and speaks the update on each
wake from that output, re-arms with `since` = the `ts` of the line just spoken, and on the `standup`
wake speaks the First-Run Report instead. On a runtime without background execution, Sam runs `tool
run-progress show --client <slug>` on every Boss turn while the run is in flight and speaks the same
shape. Record `first_run_last_stage` (the last stage index spoken, 0–6) on `automation_manifest.md`.

run_progress.jsonl line, written by `tool run-progress --pipeline {setup-root}/daily-content-pipeline append` (07 schema, one object per line,
append only):

```json
{"ts": "2026-09-12T08:14:03+07:00", "client_slug": "acme", "run_id": "2026-09-12-acme-1",
 "stage": "find_people", "stage_index": 2, "status": "done",
 "calls_done": 6, "calls_planned": 45,
 "counts": {"search_posts": 34, "group_posts": 0, "groups_found": 0, "groups_readable": 0,
            "groups_no_access": 0, "groups_monitored": 0, "groups_not_selected": 0, "groups_paused": 0, "people_found": 57,
            "leads_hot": 1, "leads_warm": 3, "leads_watch": 2},
 "platforms": {"facebook": "running", "instagram": "running", "x": "tripped:rate_limit"},
 "eta_low_at": "2026-09-12T08:40:00+07:00", "eta_high_at": "2026-09-12T09:00:00+07:00",
 "note": ""}
```

`status` is `done` at a boundary, `stalled` when a platform's job produced no result within its TTL,
`aborted` when the run stopped early (say why in `note`). `platforms` values: `running | done |
skipped:{reason} | tripped:{signal} | web_only`. The wait helper matches `client_slug` and `ts` only.

### Fixed order per platform: search/feed, then people, then depth, then intent/comments/replies

Within a single platform, the order below is fixed because each step narrows and ranks what the
next step touches. Do not reorder, parallelize, or skip a step within a platform to save budget —
skip whole groups/profiles instead once that platform's budget runs out. Across platforms, the three
still interleave round-robin per the rule above; a platform never gets two consecutive jobs while
another platform has a pending step.

**Facebook**

1. **FEED FIRST.** `fb.search.posts` with `search_url =
   https://www.facebook.com/search/posts/?q=<discovery term>`. This is Facebook's own global Posts
   search — no group membership needed, and it is the fastest first signal of who is using
   in-market language right now.
2. **THEN PEOPLE.** `fb.people.search` with `query = <discovery term>` (the URL Facebook itself
   renders: `https://www.facebook.com/search/people/?q=<discovery term>`). Returns `ProfileSummary`
   rows (`id`, `name`, `url`, `subtitle`, `mutual_friends`, `industry_hint`) — people, not posts, so
   there is no post text to read. Every row goes through Stage 10 as a PERSON lead candidate: the
   `subtitle`/work line is the qualifying signal (e.g. "Realtor at ..." when the client sells to
   realtors), and classification uses `subtitle` + `industry_hint` + `name`/`url` only. Step 1 of the
   Lead Qualification Rule decides: a `subtitle` that matches a line of `buyer_profile.types` is
   `fit = high`, and with no stated need that is `intent = none` → `warm` per the matrix
   (`playbooks/LEAD_QUALIFICATION_RULE.md`); no match stays lower per the same rule.
   Qualified rows go to `tool crm-store ... lead capture` like every other lead, tagged
   `source:people_search` plus `kw:{term}`.
3. **THEN GROUPS.** `fb.groups.search` with `query = <discovery term>`. Keep a group when
   `privacy == "public"` OR `viewer_join_state == "MEMBER"` (the account already belongs, so it can
   read a private group) — these are `groups_readable`. A private group the account is not in
   (`viewer_join_state` `CAN_REQUEST` / `REQUEST_TO_JOIN`, or `privacy == "private"` with any other
   join state) is `groups_no_access`: registered `state: no_access`, shown to the Boss as "join it
   yourself in your own session if you want it monitored", and never scanned, never joined, never
   requested (join boundary, `playbooks/skills/lead-engine/safety.md`). If `privacy` is empty/unknown,
   one `fb.group.posts` call with `max_pages: 1` decides — posts come back → `readable`; an access
   wall, or empty with `stopped_because` naming access or login → `no_access` — or skip the group when
   the call budget is tight, never guess it readable. Score every readable group with the Group
   Potential Rule below, preferring names/snippets that match the client's target location, then
   register it: `tool source-registry add --client <slug> --platform facebook --source-type group
   --origin discovered --url <u> --name <n> --member-count <m> --privacy <public|private> --state
   <active|not_selected|no_access> --potential <high|medium|low> --reason "<one line>"`. Skip a group
   already in `private_data_sources`, and skip a group this pass already scanned in the last 7 days.
   the former per-month group shortlist file is retired — the source registry is the only
   store; see `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` for the registry fields.

   **Stop the group search at 20.** Step 3 stops issuing `fb.groups.search` calls as soon as this
   run has registered 20 groups with `state: active` (counting groups registered this run only):
   the remaining discovery terms are not spent, and the pass goes straight to step 4. Raw search
   results may be far more than 20 — the Group Potential Rule filters them first, and only `active`
   ones count toward the 20. Joined-places discovery (`playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step
   12A) is not capped this way: it registers everything it finds in one read, and the Group sweep
   before optimisation order below is what spreads the digging across days.

### Group Potential Rule

For every readable group (public, or private where the account is a member — `groups_readable`),
write `potential: high | medium | low` and one-line `potential_reason`, using the group's name,
snippet/description, member count and the client's `buyer_profile` (`types`, `location`, `sells_to`):

- **high** — the client's buyer types gather there (a trade/role group matching a `types` line, e.g.
  realtors of Orange County for a client that sells to realtors; first-time-buyer groups for a
  realtor client), OR a local community group inside `location` where those buyers turn up now and
  then (neighbourhood, buy/sell/marketplace, parents, expat or Vietnamese-community, city or county
  groups, local business networks).
- **medium** — buyer types match but the location is unclear or wider than `location`; or a community
  group next to (not inside) the location; or a large mixed group where buyers appear occasionally.
- **low** — neither buyer types nor location fit (another country or language irrelevant to the
  client, hobby or meme groups, groups that are only competitors advertising, job boards); or the
  group cannot be read.

`high` and `medium` are monitored automatically (`state: active`); `low` is recorded as `state:
not_selected` with the reason so it is never re-judged the next day; a private group the account is
not in is `state: no_access` (listed in the report as "worth joining", never joined by the agent).
Judge from what the row shows; when nothing shows a fit, use `low`. No human confirms any of this;
the Sources page shows the active list with Pause/Resume.

### Group sweep before optimisation

**Group sweep before optimisation.** Until every monitored group has been scanned at least once,
each run digs the NEXT 20 never-scanned groups (newest registered first), not the best performers —
the point is to see all of the client's groups before deciding which are worth repeating. Only when
no monitored group has `scans == 0` does the plan switch to the performance order (most leads
across the last 3 scans first, then longest-unscanned). A run with fewer than 20 never-scanned
groups left fills the remaining slots from the performance order, so a run is never half empty.

`tool source-registry plan` implements this — no prose decides it. New ordering: `scans == 0`
first, `added_at` newest first among them; then the existing order (sum of `leads_recent` desc →
longest-unscanned → member count desc → uid). The result carries `"phase": "sweep" | "optimize"`
(`sweep` while any active group still has `scans == 0`) and `"unscanned_remaining": <N>` (active
groups with `scans == 0` after this plan's cut); `--max` default stays 20. The run reply and the
progress line say which phase the client is in while `phase == "sweep"`: "đang khám phá group: còn
{unscanned_remaining} group chưa quét lần nào".

4. **THEN IN-GROUP.** Scan the groups from `tool source-registry plan --client <slug> --platform
   facebook --max 20` — up to 20 monitored groups (`state: active`), in the order the Group sweep
   before optimisation rule above returns; whatever does not fit the 20 rolls to the next run
   automatically. For each planned group, private or public alike, `fb.group.search_posts` with
   `group_search_url = <group_url>/search/?q=<intent term>`. Intent terms — not the discovery term —
   come from `tool source-keywords ... plan`; seed the group's bank first if it is empty (`seed
   --industry --market --lang`, plus the client's setup seed file when one exists). `groups_no_access`
   and `not_selected` groups are never planned. After each group scan, record it: `tool
   source-registry record --client <slug> --run <run_id> --url <group_url> --leads <n>` (n =
   hot+warm+watch captured from that group this scan) — this is what re-ranks the plan for the next
   run.

**Instagram**

1. **SEARCH.** `ig.search.posts` with `query = <discovery term>` against Explore search. A term
   written as `#tag` is passed to the capability as-is, not stripped or re-encoded. Fastest first
   signal of who is posting in-market language right now.
2. **THEN PEOPLE.** `ig.people.search` with `query = <discovery term>`. Top matches only — the
   capability has no pager and returns at most 50 rows. Every row goes through Stage 10 as a PERSON
   lead candidate exactly like Facebook's people search: classify off the bio/handle text. Step 1 of
   the Lead Qualification Rule decides: a bio that matches a line of `buyer_profile.types` is
   `fit = high` → `warm` with no stated need (`intent = none`), per the matrix
   (`playbooks/LEAD_QUALIFICATION_RULE.md`). Qualified rows go to `tool crm-store ... lead capture`
   tagged `source:people_search` plus `kw:{term}`, platform `instagram`.
3. **THEN PROFILE DEPTH.** `ig.profile.posts` on the best hits from steps 1-2 (the accounts whose
   posts or bios qualified) — reads that account's recent posts for more direct-need signal than a
   single search hit gave. `ig.profile.enrich` is available for a deeper profile read when useful;
   it is not a required step and, if used, counts against this platform's budget for the run.
4. **THEN COMMENTS.** `ig.post.comments` on the best posts found in steps 1 and 3 — the comment
   thread under an in-market post is often where the actual buyer, not just the original poster,
   shows up.

**X**

1. **SEARCH LATEST.** `x.search.posts` using Latest (product `Latest`, i.e. `f=live`) with
   `query = <discovery term>` — chronological, not X's relevance ranking, so it surfaces who is
   posting right now rather than who is already popular.
2. **THEN PEOPLE.** `x.people.search` with `query = <discovery term>`. Every row goes through
   Stage 10 as a PERSON lead candidate exactly like Facebook's people search: classify off the bio
   text. Step 1 of the Lead Qualification Rule decides: a bio that matches a line of
   `buyer_profile.types` is `fit = high` → `warm` with no stated need (`intent = none`), per the
   matrix (`playbooks/LEAD_QUALIFICATION_RULE.md`). Qualified rows go to `tool crm-store ... lead
   capture` tagged `source:people_search` plus `kw:{term}`, platform `x`.
3. **THEN PROFILE DEPTH.** `x.profile.posts` on the best hits from steps 1-2. `x.profile.enrich` is
   available for a deeper profile read when useful; it is not a required step and, if used, counts
   against this platform's budget for the run.
4. **THEN REPLIES.** `x.post.replies` on the best posts found in steps 1 and 3.

`x.timeline.home` exists in the catalog but is NOT part of this pass — it is the account's own
personal feed, not a discovery surface.

### Discovery terms

Discovery terms for all three platforms — Facebook's steps 1-3, Instagram's steps 1-2, X's steps
1-2 — come from a new keyword-bank kind, not from the source-keywords intent bank Facebook step 4
uses:

```sh
<bridge> tool public-keywords --pipeline daily-content-pipeline --client {slug} plan --kind discovery
```

`--kind discovery` is the `community_discovery` kind (default 3 terms), shared across all three
platforms. Add one by hand with
`<bridge> tool public-keywords --pipeline daily-content-pipeline --client {slug} add --group community_discovery --term "..."`.

The `record` verb has no `--kind` flag and requires a verdict. Record outcomes the same way the
public-keywords bank always does, with `urls` = posts + people rows (+ groups, Facebook only) found
for that term across whichever platforms ran it, and `ideas` = leads captured:

```sh
<bridge> tool public-keywords --pipeline daily-content-pipeline --client {slug} record --json '{"{term}":{"verdict":"useful|used|weak|retry_later","urls":N,"ideas":M}}'
```

Verdict is `useful` when the term produced ≥ 1 lead, `used` when it produced results but no lead,
`weak` when it produced nothing, and `retry_later` when a safety trip cut the term short.

### Every post and person row goes through Stage 10 immediately

Every post record this pass returns on any platform — from a feed/search search, from the Facebook
group list, from Facebook in-group search, from Instagram profile depth or comments, or from X
profile depth or replies — and every person/profile row a people search returns on any platform,
goes through this stage's Detection Workflow and then straight to `<bridge> tool crm-store ... lead
capture` (see "Every lead also becomes a CRM contact" below) with `platform` set to `facebook` |
`instagram` | `x`, EVEN THOUGH the Facebook group it came from might not (yet) be registered
`state: active`. The pass acquires and captures; it does not wait on a group's monitoring state
first.

### Step 5 — will this thread's repliers be the client's buyers?

Every post this pass judges — and every other post-level judgement in this stage's Detection
Workflow, not only the Social Discovery Pass — also runs Step 5 of
`playbooks/LEAD_QUALIFICATION_RULE.md`, right after Steps 1-4 decide the post's own author. Step 5
asks one question about the THREAD, not the author: will the people who answer this post be the
client's buyers? The rule returns `comment_source: likely|unlikely`, `types_match`, and a one-line
`reason`.

A `likely` thread is RECORDED, never harvested in the run that found it — nothing under it is read,
fetched, or captured beyond what the post-level judgement already saw:

```sh
<bridge> tool source-registry --pipeline daily-content-pipeline --client {slug} discovered add \
  --json '{"platform":"facebook","post_url":"...","community":"...","author_line":"...",
           "excerpt":"...","comment_source_reason":"...","types_match":true,
           "comment_count":N,"run_id":"..."}'
```

`id` is derived as a stable hash of `post_url`, and the verb dedupes on it: a source already known
just gets `last_seen_at` bumped, `status` untouched, never reset. Fields and the full registry
schema (including `status: new|approved|harvested|dismissed`, which nothing here ever advances past
`new`) live in `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`. An `unlikely` verdict is judged and
dropped — only `likely` is persisted. See "Harvest a discovered thread (on the Boss's order only)"
below for what happens to a recorded source, and only on explicit order.

### Lead target: a floor, not a stop

FIRST RUN: **minimum 10 leads across all platforms combined.** Reaching 10 does not end the run —
keep working down the ranked candidates on every enabled platform until the day's budget (above) is
spent on each; more is always better than exactly 10. If the combined budget runs out below 10, say
so plainly in the report; monitored groups the plan could not fit into this run's cap of 20 simply
roll into the next day's registry plan (`tool source-registry plan`) automatically, nothing is lost.

DAILY companion pass: no fixed floor on any platform — it is a light top-up, bounded by each
platform's own smaller budget above; groups left over from a prior day's plan carry over to this
run's plan the same way — nothing waits on a Boss answer.

### Budget (owner-approved, inside the safety envelope; serial, paced by the collector)

The platform table above carries the exact ceilings. Which tier applies is decided PER PLATFORM by
that platform's own `{platform}_discovery_first_pass_done` field on the Client Intelligence Profile,
not by the automation run's own number: **FIRST RUN** applies to a given platform whenever its field
is not yet `true` — i.e. this is the first time this pass has ever executed on that platform for
this client, whether that happens to be run 1, run 2, or run 20 because the platform was connected
late — and the agent sets that platform's field `true` immediately after this pass completes for it.
**DAILY** applies to every pass after that, per platform. A platform on FIRST RUN and another on
DAILY in the same run is normal and expected (e.g. Facebook connected weeks ago, Instagram connected
today) — each platform's tier is independent.

These numbers are ceilings from `playbooks/skills/lead-engine/safety.md`, which carries the full
envelope (pacing, serial-only, the trip rule, round-robin). Never raise them without the human's
explicit approval.

**Pacing Rule (all platforms, both tiers).** The only spacing between requests is the collector's
own random delay: every discovery job carries `"pacing": {"min_delay_seconds": 5,
"max_delay_seconds": 10}` — a random 5–10 second pause before each page, scroll or request inside
the job — plus the extension's poll interval between jobs. The agent submits the next job as soon as
it has read the previous job's result for a trip signal; it never sleeps, waits, or schedules its own
gaps between jobs, never spreads a pass over hours, and never lowers the delay below 5 s. Round-robin
order (Facebook → Instagram → X) and the per-platform ceilings are unchanged; with this pacing a
FIRST RUN on all three platforms takes about 30–45 minutes of collector time (ETA Rule), not hours.
What protects the account is the trip rule, not slowness: the first checkpoint, rate-limit warning
or logged-out signal stops that platform for the day.

**ETA Rule (never a default number).** Before dispatching any run, and again in every progress
update, compute the expected duration from the plan, never from habit:

- `calls_planned` = the sum, over platforms whose `{platform}_lead_source` is `enabled`, of that
  platform's ceiling for its tier (Facebook: discovery calls + groups planned × terms per group, up to 69 FIRST RUN / 43 DAILY with 20 groups; Instagram 12 / 4; X 12 / 4), plus one call per
  custom or private source scheduled this run, plus one per `web_only` platform being re-probed.
- One collector call ≈ `max_pages × (7.5 s average delay + ~5 s page load)`, capped by the 60 s
  capability timeout → use 40 s (low) and 60 s (high) per call.
- `eta_low = calls_planned × 40 s + 10 min`; `eta_high = calls_planned × 60 s + 15 min` (the fixed
  part covers keyword planning, qualification, report rendering, upload, notification).
- Worked examples (20 monitored groups planned): three platforms on FIRST RUN → up to 93 calls → 72–108
  min. Facebook only, FIRST RUN → up to 69 calls (9 discovery + 20 groups × 3 terms) → 56–84 min. DAILY on
  three platforms → up to 51 calls → 44–66 min. Facebook only, DAILY → 43 calls → 39–58 min. Fewer groups
  planned means fewer calls — always read the plan. No platform enabled → 10–15 min.

Once `calls_planned` is summed, run `tool run-progress eta --calls <calls_planned>` and read the
numbers from its output — never compute `eta_low`/`eta_high` by hand. Speak it as a window of clock
times in the Boss's language ("bắt đầu 08:00, dự kiến xong 08:40–09:00"), quote the tool's
arithmetic summary verbatim (its one-line stderr summary, e.g. "45 lượt gọi × 40–60 giây + 10–15
phút"), and record `run_calls_planned`, `run_eta_low_min`, `run_eta_high_min`, `run_eta_at` on
`automation_manifest.md` at dispatch straight from that output. The background wait's `--timeout`
is the tool's `wait_timeout_s` field, passed directly to `wait_for_run --timeout` (equivalent to 1.5
× eta_high, minimum 1800). After each stage, recompute by calling `tool run-progress eta` again with
the calls remaining; a platform that trips mid-run removes its remaining calls from the plan.

### Safety trip is per platform, and unforgiving

The first checkpoint, rate-limit warning, or logged-out signal on a platform stops THAT PLATFORM for
the day — not the other two, and not the whole run (see the round-robin rule above). After every
single job in this pass, the agent must read that job's result for those signals BEFORE submitting
the next job on that same platform. This holds inside each platform's FIRST RUN or DAILY budget
exactly as it holds everywhere else in this playbook.

### Scanning and monitoring need no approval (Facebook only)

Scanning a readable Facebook group in this pass (public, or private where the account is already a
member) needs no per-group human approval — see the join boundary in
`playbooks/skills/lead-engine/safety.md` and the reconciliation paragraph in
`playbooks/PRIVATE_SOURCE_GATE.md`. Monitoring needs none either: the Group Potential Rule above
scores every readable group the moment this pass finds it, and `high`/`medium` potential is
registered `state: active` — standing daily monitoring — automatically. `low` potential is registered
`state: not_selected` with the reason, so it is never re-judged the next day. A private group the
account is not a member of is registered `state: no_access` and listed in the report as worth
joining — the agent never joins it. There is no shortlist `status`/`decision` pair any more, no
no review-state field, and no close gate: neither the Setup Flow nor a scheduled
run waits on a group decision, because there is no group decision to wait on. The human's only lever
is pausing or resuming an active group on the Sources page (`playbooks/02_PRIVATE_SOURCE_SETUP.md`).
A "no" to custom URLs at Setup Flow step 5 is a different question and has no bearing on monitored
groups. Instagram and X have no group concept, so this whole boundary applies to Facebook only.

Groups this pass finds should also be registered in the shared source registry as readable groups
(`tool source-registry register`, existing shape) so other clients' passes reuse the notes instead of
rediscovering the same group cold.

### Report section: "Social Discovery Pass"

Every run of this pass gets its own named report section (translate the title naturally in a
non-English report; English default below), with one row per platform that had at least one enabled
step this run:

- discovery terms used;

**Five result types (never merged).** Every progress update, the First-Run Report, the run reply
and the INTERNAL_REPORT "Social Discovery Pass" section list these separately, in this order, each
with its own number; a zero is printed as 0 and a type that does not apply to a platform (Instagram
and X have no groups) is printed as "—":

1. Bài từ Facebook Search theo từ khóa — `feed_posts_found` (Instagram/X: `posts_found` from search);
2. Bài/clip quét bên trong group — `group_posts_found` (Instagram/X: `depth_posts_found` from
   profile depth plus comments/replies);
3. Group ứng viên tìm thấy — `groups_found`, split into `groups_readable` (public, or private with
   the account already a member) and `groups_no_access` (private, account not a member — listed for
   the Boss to join, never scanned);
4. Group đang theo dõi (agent tự chọn) — `groups_monitored` (state: active, this run / total), plus
   `groups_not_selected` and `groups_paused`;
5. Lead đạt luật — `leads_found` with hot/warm/watch, plus `leads_locked`.

- budget used (calls spent / calls available, for that platform's tier this run);
- trip status (`clean`, or the exact safety trip that stopped that platform for the day).

```text
Social Discovery Pass
Facebook  — Terms: 3 · Feed posts: 3 · In-group posts: 14 · Groups: 12 found (7 readable, 5 no-access) · Monitored: 6 (2 not selected) · Leads: 19 (9 hot, 6 warm, 4 watch) · Locked: 0 · Budget: 20/21 · Trip: clean
Instagram — Terms: 3 · Posts: 8 · Depth posts: 6 · Groups: — · Monitored: — · Leads: 7 (2 hot, 3 warm, 2 watch) · Locked: 0 · Budget: 11/12 · Trip: clean
X         — Terms: 3 · Posts: 11 · Depth posts: 9 · Groups: — · Monitored: — · Leads: 5 (1 hot, 2 warm, 2 watch) · Locked: 0 · Budget: 12/12 · Trip: clean
```

## Definitions

### Lead

A lead is a person, account, post, comment, or thread that the "Lead Qualification Rule (Fit ×
Intent)" section below scores as `hot`, `warm`, or `watch` against the client's `buyer_profile`. The
rule — not a shown need — is the test: **a right-type person with no stated need today is still a
lead** (`fit = high`, `intent = none` → `warm`). Read `playbooks/LEAD_QUALIFICATION_RULE.md` in full
for the actual decision logic; do not use a "must show a direct or indirect need" bar anywhere in
this system — that framing is retired.

What the rule scores, in the shape of the signals it names:

- **who they are (fit)** — `person_type` matched against `buyer_profile.types`, decided first and
  always, need or no need;
- **why now (intent)** — a stated or implied friction from `buyer_profile.why_they_need`: asking for
  a provider/quote/recommendation, describing a problem the offer solves, comparing options, a life
  event or new asset/job/deadline that creates the friction — these raise `intent` from `none` to
  `implied`/`explicit` and can move a `warm` lead to `hot`, but their absence never removes the lead;
- **competitor or noise** — Step 2 of the rule overrides the matrix: a same-market seller is
  `competitor`, not a lead; someone who cannot buy or explicitly rejects the offer is `none`.

A lead is a signal for the human to review. It is not permission for the agent to contact the person.

### Competitor

A competitor is any person, account, company, creator, product, community, content asset, or alternative solution that competes for the same target audience, buying intent, attention, or trust.

Competitor types:

- `direct_competitor`: offers the same or very similar product/service.
- `indirect_competitor`: solves the same problem with a different service, product, tool, or method.
- `adjacent_solution`: serves the same pain point before or after the user's offer.
- `attention_competitor`: attracts the same target audience even if it does not sell the same thing.
- `authority_or_kol_competing_for_trust`: owns trust, education, or recommendation power in the same audience.

Competitor posts matter because they often attract the same people the user wants to help. The user should either learn from the signal or appear in the discussion with a useful, natural comment.

## Scan Depth Contract

Lead and competitor extraction should happen inside the normal data collection pass:

```text
research/source scan -> data points -> leads -> competitors -> ideas -> best idea -> draft -> report
```

Do not run a second scan just for lead/competitor detection unless:

- the human explicitly asks for a deeper lead/competitor pass;
- the first scan failed or produced too little usable data;
- a saved schedule or config says a deeper first pass is allowed.

Private data source depth:

- First lead/competitor pass for a client/source set: 10 scrolls per approved source.
- Recurring daily run: 5 scrolls per approved source.
- Use `collector_config.scroll_delay_seconds`, defaulting to about 5 seconds.
- Respect `collector_config.max_scrolls_allowed`, account-safety limits, rate-limit warnings, session-expired states, and platform warnings.
- If Local Collector config exists and has lower safety settings, obey the safer lower value and record why coverage is lower.
- If the human configured a higher number, cap it at Stage 8 safety limits unless the collector explicitly supports a safe higher discovery mode.

Human-facing disclosure when scanning private groups/sources:

```text
I will go through each approved group/source one by one and scroll {N} times per source. For the first lead/competitor pass I use 10 scrolls when safe; for normal daily runs I use 5 scrolls. I read this from the Local Collector configuration when available.
```

## Lead Qualification Rule (Fit × Intent)

`playbooks/LEAD_QUALIFICATION_RULE.md` is the rule of record for qualifying every item this stage
sees — posts, comments, captions, and people-search rows alike. This section is a faithful summary;
read the rule file in full before scoring, and re-read it after any edit to it (the file's own edit
policy requires re-running `playbooks/tests/lead-rule` and attaching the report).

The rule answers three questions in order for ONE item against ONE client's `buyer_profile`, then
reads the decision off a matrix:

**Step 1 — WHO is this person? (fit, decided first, always).** Name `person_type` from every clue in
the item (what they do, own, or run; where they are; the community; language). Test membership
against THIS client's own `buyer_profile.types` list (never another client's): the category before
"e.g." is the test, the examples after it are illustrative, never a closed list — "no evidence of
doing X" is forbidden as a reason. `fit = high` when `sells_to_match` is a real match and, where
location matters, the person is inside or arriving into the service area. `fit = medium` for an
adjacent profession/situation, or an exact match with unclear location. `fit = low` for no match,
evidence of being outside `sells_to`, or a hedged/different situation.

**Step 2 — Competitor or noise? (decided before intent, every time).** `competitor`: sells the same
thing to the same market (a same-market seller is a competitor even with a shown need, unless
`types` names that profession as a customer). `none`/noise: cannot buy in any reading, explicitly
rejects the offer, or is stale — a high-fit person who rejects the offer is `none`, not `warm`.

**Step 3 — WHY NOW? (intent).** Name the friction in `why_they_need` the content shows, or intent is
`none`. `intent = explicit` when they ask for, compare, or complain about exactly this kind of offer.
`intent = implied` when a named new thing (asset, dependent, job, address, business, deadline) creates
that friction. `intent = none` when nothing points to a need today — routine work, true on any
ordinary working day, stays `none`.

**Step 4 — Decision (the matrix, the same for every industry).**

| fit | intent explicit | intent implied | intent none |
|---|---|---|---|
| high | **hot** | **hot** | **warm** |
| medium | **warm** | **watch** | none |
| low | **watch** | none | none |

Compute fit and intent independently, then look up that cell. `competitor`/`none` from Step 2
override the matrix outright.

**A right-type person with no stated need today is still `warm`, and is still captured** — the
high-fit/no-intent cell is not a rejection, it is the normal shape of a good customer who has not yet
shown a trigger. Never drop a person of the right type for lacking a stated need.

**Step 1 runs first on every surface, not only in-group search**: the feed/scroll pass classifies
each post by the author's `person_type` before anything else, and every people-search row (Facebook,
Instagram, X — see "Social Discovery Pass" above) is judged the same way, off whatever author line is
available (subtitle/bio/headline).

**Keyword channels.** Discovery terms for the feed/people-search legs are unchanged
(`community_discovery` kind). In-group search terms follow the per-channel table in
`playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`'s keyword-bank section: role and product/stage terms are
primary (what the person calls themselves, what they say about their work); intent terms are used
only when anchored to the offer or the role, never as a bare need phrase.

Required lead fields:

- **name — the author's name, copied verbatim from the collected record.** The collector stores it at
  `records.items[N].actor.name` (and `.replies[M].actor.name` for a comment), and it is on disk in every
  post, group-search and comment capture. Carry it into the ledger row's `name` field. A post scan always
  has it, so a lead row without a name means the carry was skipped, not that the person is anonymous.
  Never put Facebook interface text here — `Top contributor`, `Rising contributor`, `Verified account`,
  `Anonymous participant`, `See more`, a group's name, or a url. The CRM capture refuses all of those and
  the contact stays honestly nameless. Omit `name` only for a lead added from a bare post/reel/video url
  before anyone enriched it; that case falls back to the profile handle for display.
- lead label or safe descriptor;
- source/platform;
- source type: public | private;
- profile URL when visible and safe;
- post/current URL when available;
- **a URL field is either a real URL or absent — never a placeholder.** Writing `unavailable`,
  `masked_per_retention_rule`, `n/a`, `none` or `-` into `profile_url` or `post_url` used to make
  that word the person's identity, so on this install eleven different people merged into one
  contact and four more into another. Omit the field instead. An in-page relative href
  (`/groups/<gid>/posts/<pid>/`) is fine: capture resolves it against the platform. A row that
  ends up with no URL at all is still captured as long as it carries what was observed (summary,
  evidence, why it matters) — the person is anchored on the sighting rather than discarded;
- **one post URL must mean one person.** When the row has no `profile_url`, do not reuse a
  section or index page (`/ask-an-expert/state/california/`) as the post URL for several
  different people — capture keeps them apart by their own evidence, but the ledger reads as
  though four homeowners shared one thread. Give each person the URL of the thing they wrote;
- captured at;
- emails (optional) — email addresses extracted from the page content the collector captured (visible text + `mailto:` links); present when the captured content contains them; empty/absent otherwise;
- phones (optional) — phone numbers extracted from the same captured page content (visible text + `tel:` links), digits normalized (E.164-style with a leading `+` when a country code is present); present when the captured content contains them; empty/absent otherwise;
- evidence snippet or safe summary;
- person_type: what this person is, in a few words (Lead Qualification Rule Step 1);
- sells_to_match: the closest line of this client's `buyer_profile.types`, or none;
- fit: high | medium | low;
- fit_reason: one line;
- intent: explicit | implied | none;
- intent_reason: one line;
- lead type: direct_need | indirect_need | pain_signal | buying_trigger | objection | comparison | complaint | adjacent_need — the evidence label, kept alongside fit/intent, not replaced by them;
- lead level: hot | warm | watch — derived from fit × intent (Lead Qualification Rule Step 4), never chosen independently;
- related offer;
- related pain point;
- confidence: high | medium | low;
- suggested next action for the human;
- suggested value-first comment;
- outreach/compliance note.

Required competitor fields:

- competitor name/page or safe descriptor;
- competitor type;
- platform/source;
- profile URL when available;
- post/current URL when available;
- captured at;
- audience overlap;
- offer/positioning;
- content theme or hook pattern;
- engagement or comment signal;
- threat/opportunity level: high | medium | low;
- what the user can learn;
- suggested value-first comment;
- monitoring action.

## Opportunity Scoring

Lead level is not scored qualitatively — it is derived from the Fit × Intent matrix
(`playbooks/LEAD_QUALIFICATION_RULE.md` Step 4; see "Lead Qualification Rule" above). `confidence`
(high | medium | low) stays a separate field, judging how clearly the item supports the `fit_reason`
and `intent_reason` given, not the lead level itself.

Competitor score dimensions:

- audience overlap;
- engagement quality;
- repeated pain points in comments;
- positioning strength;
- freshness;
- content pattern usefulness;
- strategic threat;
- opportunity for the user to add a better, clearer, or more helpful perspective.

## Audience Value-First Opportunity Rule

Lead and competitor intelligence must produce useful audience-facing ideas, not direct praise for the user's product/service.

When a lead or competitor signal becomes an idea, best idea, suggested comment, script, blog, caption, or recommendation, it must state:

- the audience pain point or confusion;
- the viewer value / lesson;
- the source signal;
- the non-promotional angle;
- why it helps the audience;
- the soft business relevance to the user's offer.

Do not convert competitor positioning into `the client should out-position them`, `our product is better`, `{Client Product} wins`, `choose us`, or similar direct promotional framing. If the idea cannot teach the audience something useful without selling, reject it as `promotional_not_value_first`.

Bad competitor-derived idea:

```text
MiniMeo out-positions competitors by selling without selling across multiple platforms.
```

Better competitor-derived idea:

```text
How small brands can test whether their "selling without selling" message is actually clear before pushing it across every platform.
```

Use qualitative labels if numeric scoring would slow the run:

```text
high | medium | low
```

## Comment Drafting Rules

For every lead and competitor opportunity in the HTML report, draft one short comment the human can copy.

The comment must:

- use the same language as the post or thread;
- respond to the actual context, not a generic template;
- provide a useful insight, question, clarification, checklist item, or perspective;
- sound like a real person, not an AI assistant;
- be short enough to paste into a social thread without looking like a mini blog post;
- avoid direct selling, self-promotion, `DM me`, `message me`, `inbox me`, `book a call`, `reach out to start`, `we can help`, or service pitches;
- avoid attacking or undermining a competitor;
- avoid guarantees, regulated claims, legal/medical/financial advice, or unsafe instructions;
- avoid over-polished phrasing, generic motivational language, and obvious AI cadence;
- avoid bullet lists unless the platform/context naturally uses bullets.

The comment may include one or two tiny natural imperfections when appropriate, such as a casual spelling choice, a missing accent, or a small human-sounding typo. This is allowed because a slightly imperfect helpful comment can feel more trustworthy than perfectly polished AI-style text.

Do not force typos. Never make the user look careless, rude, uneducated, unprofessional, or unclear. The comment must remain easy to understand.

Good comment pattern for a lead post:

```text
This usually gets easier if you separate the urgent question from the long-term decision. First check what has to happen this week, then compare options after that. A lot of people accidentally mix those two and overpay or choose too fast.
```

Good comment pattern for a competitor post:

```text
One thing I like about this topic is that the "right" answer depends a lot on timing. The same advice can be great before renewal and pretty bad after a notice already arrives.
```

Bad comments:

```text
We help with this. DM me.
```

```text
As an expert in this industry, I strongly recommend scheduling a consultation today.
```

```text
This competitor forgot to mention that our service is better.
```

## HTML Report Contract

The report must include one section named:

```text
Lead & Competitor Opportunities
```

If the report is in another language, translate the title naturally. This is the only Lead & Competitor Opportunities section — there is no public/private pair to keep in sync. Group its entries by source (Google/Web, Facebook, Instagram, X, industry sites, custom URLs — only the ones that ran), and keep the source visible next to each entry so the reader can always tell where it came from.

For each opportunity, include:

- type: lead | competitor | both;
- classification, such as hot lead, warm lead, direct competitor, indirect competitor, attention competitor;
- source/platform;
- post/current URL as a visible link when available;
- profile/account URL when visible and safe;
- safe context summary;
- why this matters;
- suggested human action;
- copy-ready comment;
- a real working copy button for the comment.

Copy buttons are allowed because they perform a real local browser action. They must copy only the suggested comment text. They must not imply that the comment will be posted automatically.

Example HTML behavior:

```html
<button type="button" class="copy-comment" data-copy="Helpful comment text here">Copy comment</button>
<script>
document.querySelectorAll('.copy-comment').forEach(function (button) {
  button.addEventListener('click', async function () {
    var text = button.getAttribute('data-copy') || '';
    try {
      await navigator.clipboard.writeText(text);
      button.textContent = 'Copied';
    } catch (error) {
      var area = document.createElement('textarea');
      area.value = text;
      document.body.appendChild(area);
      area.select();
      document.execCommand('copy');
      document.body.removeChild(area);
      button.textContent = 'Copied';
    }
  });
});
</script>
```

If no opportunities were found, the report must distinguish:

- no leads/competitors found after scanning;
- not scanned because a source is not yet connected;
- not scanned because source/session was unavailable;
- only the connected sources ran, so coverage is limited until the rest connect.

## Storage Contract

Store lead and competitor opportunities in both the existing lead/competitor logs and the unified opportunities ledger when possible.

Recommended unified ledger:

```text
history/YYYY-MM/lead_competitor_opportunities.jsonl
```

Recommended JSONL fields:

```json
{
  "date": "YYYY-MM-DD",
  "client_slug": "client",
  "opportunity_type": "lead",
  "classification": "hot_lead",
  "source": "Facebook Group",
  "source_type": "private",
  "platform": "facebook",
  "name": "Rick Silva",
  "profile_url": "https://...",
  "post_url": "https://...",
  "emails": ["name@business.com"],
  "phones": ["+14155550100"],
  "captured_at": "ISO-8601",
  "safe_context_summary": "Short summary",
  "evidence_snippet": "Short visible snippet if safe",
  "why_it_matters": "Reason",
  "related_offer": "Offer",
  "related_pain_point": "Pain point",
  "confidence": "medium",
  "suggested_action": "Human reviews and decides whether to comment",
  "suggested_comment": "Value-first comment",
  "comment_language": "en",
  "comment_style_notes": "natural, short, no direct pitch",
  "status": "needs_review"
}
```

**`name` is required whenever the scan saw one, and a post scan always sees one.** The author's
name is on the page next to the post; carrying it is the difference between a CRM row a human
recognises and a row labelled with a database id. Copy it verbatim from the collected record
(`actor.name` / the post author) into `name`. Rules:

- Do NOT invent, translate, or reformat it. A business page's name is its name.
- Do NOT put Facebook interface text in this field: `Top contributor`, `Rising contributor`,
  `Verified account`, `Anonymous participant`, `See more`, a group's name, or a url. The capture
  refuses all of those and the contact stays nameless, which is the honest outcome.
- Leave `name` out only when there genuinely is none — a lead added from a bare post, reel or
  video url before anyone enriched it. That case falls back to the profile handle for display.
- `emails` and `phones` are the same contract: if the collector found them, carry them. They are
  usually absent until an enrichment pass runs, and that is fine — but a dropped address is a
  person the system can never write to, because the send gate needs one.

The optional `emails` and `phones` arrays are additive: they are populated when the collector's structured extractor finds contact details in the captured page content, and stay empty/absent otherwise. Ignoring them keeps existing behavior unchanged. They do not authorize any auto outreach; the send/act gates (approval required) still apply, and the operator's own credentials/secrets are never read or transmitted.

Do not store unnecessary personal data. Keep safe summaries and source URLs. The human can inspect the original post in their logged-in session when needed.

### Every lead also becomes a CRM contact, the day it is detected

The ledger above is the audit trail. It is not where a lead lives. A person who publicly asked for
what this client sells is a lead at the coldest end of a ladder — stranger, cold, warm, engaged — and
the CRM is the thing built to walk them up it. Leaving them in a file nothing reads means no dossier,
no timeline, no follow-up, and the same person rediscovered as "new" next week.

After writing the ledger, capture the run's leads in one call:

```sh
<bridge> tool crm-store --pipeline daily-content-pipeline --client {slug} \
  lead capture --file history/YYYY-MM/lead_competitor_opportunities.jsonl
```

It reads the same JSONL this stage just wrote, so there is nothing to reformat. Per lead it creates a
contact keyed on the profile URL (or, when there is no profile, on the post URL as a content seed),
files the post as a dated evidence hook, tags it `lead`, `lead:{hot|warm|watch}`, `source:lead_scan`
and `kw:{term}` when a search term found them, and writes one `lead_detected` activity. Competitor
rows are refused by name — a competitor is a business to study, not a person to nurture. Add
`--dry-run` to see the mapping without writing.

**Capture never stops at the plan's contact cap.** Free 30, Starter 500, Pro 2000, Business 10000,
Enterprise unlimited — those are the contact caps `tool crm-store` enforces per plan tier. Hitting the
cap does not stop capture: the newest leads above the cap are stored LOCKED (no detail, no email, no DM,
no campaign) rather than dropped, and a contact that has already progressed past `lead` into any `engaged+` stage
is never locked, no matter how the count moves afterward.

Locked count and the unlocked/max ratio both come from `tool crm-store ... contact lock-status` — read
it after every capture that touched the CRM, and every time a Boss-facing reply is about to be sent for
a run that touched the CRM. Never estimate either number.

**The meter (funnel moments G and H; see `AGENTS.md`, "Upsell rule").** Whenever `contact lock-status`
shows `locked > 0`, every Boss-facing reply, `INTERNAL_REPORT`, and morning brief for this client
carries this persistent one-line meter, translated naturally into the human's language, with no
cooldown — it is a plain fact, not an ask, and it repeats on every reply for as long as the count stays
above zero:

```text
{L} leads locked under {tier} — {unlocked}/{max} open
```

When `locked == 0` but `unlocked / max_contacts ≥ 0.8`, carry the approaching-cap line instead of the
locked meter (moment G):

```text
{unlocked}/{max} open contacts used; new leads may start locking
```

**First lock of the session (moment H).** The FIRST time in this session that `contact lock-status`
shows `locked` move from 0 to > 0, deliver the SELLING moment: a full `**[ACTION REQUIRED]**` upgrade
block, spoken the way a real person would offer the upgrade, in the human's own words and language —
never this template verbatim — naming the actual counts from `contact lock-status`. Tone example
(Vietnamese, tone only, not a script to paste):

```text
Em vừa đưa 214 lead mới vào CRM, 31 lead có tín hiệu tốt. Gói Free đang mở 30 contact, 184 lead còn lại
đang khoá chi tiết, chưa gửi mail hay nhắn tin được. Mở gói Starter thì 500 contact mở ngay, không cần
quét lại.
```

The upgrade path is the WideCast key/plan (https://widecast.ai/#setup); the human runs
`tool entitlement refresh` afterward. This is the session's one Upsell-budget use for the locked-contact
reason — every reply after that first 0→>0 transition, for the rest of the session, drops back to the
plain meter line above; it never repeats the full block for the same transition. A Free install refused
`write_actions` (group post, comment, react) is the other SELLING trigger and spends the same
once-per-session budget on its own reason. `seat_limit` and a stale-token `solo_feature_not_in_tier` on
a non-`write_actions` capability are never a selling moment (Upsell rule carve-outs).

**Show the CRM link after any scan that produced ≥ 1 lead.** In the chat reply, this stage's
lead-capture step ends with the CRM line below (Team Leader frame — this line is mirrored in
`SOLO_AGENCY_PLAYBOOK.md` "Team Leader Reply Frame"; keep both copies in sync). In
`INTERNAL_REPORT` only — NEVER in the three client-facing HTML files, the combined client
report/PDF, or the client notification (Client-Blind Deliverable rule) — the same line appears after
any scan that produced at least one lead:

```text
CRM: http://127.0.0.1:17321/ui/{client_slug}/crm — {N} liên hệ mới hôm nay, {L} đang khoá theo gói; nóng nhất: http://127.0.0.1:17321/ui/{client_slug}/contact/{lead_id}
CRM: http://127.0.0.1:17321/ui/{client_slug}/crm — {N} new contacts today, {L} locked by plan; hottest: .../contact/{lead_id}
```

A zero-lead run prints the bare CRM line (`http://127.0.0.1:17321/ui/{client_slug}/crm`, no counts).
The agent PRODUCES this link as a static string from `docs/UI_DESIGN.md`'s URL map; it never
HTTP-GETs it. Hand it per the SHOW RULE (`docs/UI_DESIGN.md` §1 principle 2, OWNER DECISION
2026-09-10): print it as text every time; on Claude Code desktop also open it in the side Browser
pane; on any other local runtime also run `open`/`start`/`xdg-open` so it lands in a real browser.
It opens directly now (`--ui-auth host` default) — no entry-link/token step, no "Locked" page.

**First run for a client that has never had a CRM.** A client set up for content only has no CRM
workspace yet, and the command will say so: `no outreach workspace under .../clients/{slug}; run
init-client first`. That is not a failure of the scan — create the workspace once and re-run:

```sh
<bridge> tool crm-store --pipeline daily-content-pipeline --client {slug} \
  --business {business_slug} --location {location_slug} init-client
```

Do this silently as part of the run; it creates empty directories and sends nothing. If `init-client`
is refused, that is a bridge/technical blocker (there is no per-plan client limit any more; the
bridge keeps only a technical ceiling far above real use) — report the exact error in
`INTERNAL_REPORT` and carry on; the leads are still in the ledger.

**Running it twice is safe and is the normal case.** The same person seen on three days is ONE
contact with three activity rows and one hook per distinct post; the temperature tag is replaced, not
accumulated, so someone who cools off stops matching a "hot leads" filter; a contact who already
became a customer is never walked back to `lead`.

**Storing is not contacting, and this changes no send gate.** A captured lead has no email, so
`channels.email.status` stays `needs_data` and drafting refuses them outright. Commenting, DMs and
mail all still require the human approval they required yesterday. What changes is only that the
person is still there tomorrow — which is the precondition for warming them at all, and the reason
the evidence hook is recorded now rather than reconstructed from memory months later.

## Harvest a discovered thread (on the Boss's order only)

A discovered source recorded above (Step 5, "likely") is a note, not a job. Nothing under it is
read, classified, or captured until the Boss explicitly orders it — naming the source in chat, or
through the dashboard Discovered tab's "Approve & harvest" button, which writes a `ui_inbox` request
(`{kind: harvest_thread, source_id}`) the Team Leader picks up
(`playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`, registry schema). The daily run never harvests —
see `playbooks/04_DAILY_SCHEDULE.md` and `playbooks/SCHEDULED_RUN_ENTRYPOINT.md`.

**Budgets.** One source per order — an order names exactly one `source_id`, never "all new ones."
Reopening the thread spends only that platform's own comments-capability budget (`fb.post.comments`
/ `ig.post.comments` / `x.post.replies`), paginated, read-only — no allowance beyond what the
platform already grants that capability.

**The job.** Both steps below are driven by `tool harvest-thread` (`--pipeline DIR --client SLUG
--id SOURCE_ID`), the purpose-built command for this exact job — do not hand-roll prepare or
ingest with other tools; it already does the dedupe/prefilter/batch and CRM-capture/status-write
correctly, including a workaround for a known `crm-store` bug (below). Both `prepare` and `ingest`
require the source's `status` to be `approved` (or `harvested`, only with `--force`, for an
explicit Boss-ordered re-harvest) — they refuse outright on `new` or `dismissed`.

1. **Prepare** — `tool harvest-thread prepare`. Reopens the post through the collector — or reuses
   comments already captured in the inbox for that post, when nothing has to be re-fetched — and
   dedupes rows by author: every comment from one author becomes one row. A code prefilter drops
   rows that are empty, emoji-only, or "following"/"bump"/tag-only. Writes the survivors to
   `history/YYYY-MM/harvest/{source_id}/batch_01.json … batch_NN.json`, 40 rows each
   (`playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`, "Discovered sources", has the full folder layout).
   If nothing is captured yet it returns `awaiting_comments` instead of batches — reopen the thread
   through the collector and re-run prepare.
2. **Classify.** For every batch file, spawn one classifier sub-agent on the LOWEST model available
   (Haiku on Claude, the smallest model on Codex) applying `playbooks/COMMENT_TRIAGE_RULE.md` — the
   batch rule of record — verbatim. One sub-agent per batch file, never one sub-agent walking every
   batch serially; batch size stays 40, never resized to save a call. Write each sub-agent's
   keep/drop verdicts to a results directory as JSON (one file per batch, rows keyed by row `id`).
3. **Ingest** — `tool harvest-thread ingest --results DIR`. Every kept author is captured as a CRM
   lead with fit/intent tags and `source:thread:{id}` (`outreach/playbooks/13_CRM_CORE.md`) — warm
   by default, `intent` only when the comment itself stated one. Competitor and noise rows are
   dropped, never captured; `medium` rows are also dropped from capture but listed in the run
   report for a human override. (`ingest` deliberately does NOT call `crm-store`'s lead-capture
   with the thread's `post_url` as every row's identity seed — a known bug in
   `contactFields()`/`addContact` merges multiple distinct commenters sharing one `post_url` into a
   single CRM contact; `ingest` attaches the post as an evidence hook separately instead. Never
   substitute a manual `tool crm-store ... lead capture` loop here — it would reproduce that
   merge.) The source's `status` then becomes `harvested`, with `harvested_at`, `harvest_job_id`
   (same value as `source_id`), `leads_added`, and `authors_seen` written back automatically by
   `ingest` (no separate CLI call — `tool source-registry discovered mark-harvested` is the
   manual-fallback equivalent, not something the harvest job itself needs to call). A `harvested`
   source is never re-harvested unless the Boss orders it again, explicitly, for that specific
   source (`ingest` refuses a second run on the same source without `--force`, same as `prepare`).

**The report.** The harvest reply and `INTERNAL_REPORT` state rows kept vs dropped (with the
`medium` list for override), the CRM link line ("Show the CRM link after any scan that produced
≥ 1 lead" above), and the source's new `status`. In chat, the reply also opens the Discovered tab
per the Answer-and-Show Rule (`SOLO_AGENCY_PLAYBOOK.md`, "Team Leader Reply Frame").

## Shared-Source Lead Collisions

When lead/competitor data comes from a SHARED source scan (records carry `source_uid`/`point_uid` and the source has multiple subscribers in `collector/source_registry.json`), the same person can surface as a lead for several clients at once — and nothing else stops two clients from both reaching out to them.

- Surface the opportunity to every subscriber client normally (each report shows what its client's audience is doing).
- In each client's `INTERNAL_REPORT`, flag any lead whose source is shared: `also surfaced for N other client(s) via shared source {uid}` — the operator decides which client (if any) acts on it.
- Never let two clients' outreach both add the same shared-source person without that flag having appeared; the operator's assignment decision is the gate.
- The flag IS the protection — it is the whole design. Making a shared third-party source `exclusive` to "prevent" collisions is FORBIDDEN: it silently abandons the shared-scan model (duplicate scans, duplicate account footprint) to solve a problem this flag already solves, and it is a scope decision only the operator may make (02's scope rule). An agent that reasons its way to exclusivity from this section has misread it.

## Completion Checklist

Before claiming lead/competitor work is complete, verify:

- Stage 10 was loaded.
- Leads include direct and indirect need signals, not only explicit "I need a provider" posts.
- Competitors include direct, indirect, adjacent, attention, or authority competitors when relevant.
- The first lead/competitor pass used 10 scrolls per approved private data source when safe, or documented why it could not.
- Recurring daily runs used 5 scrolls per approved private data source by default, or documented the configured value.
- Detection happened during the same data collection pass unless a human-approved deeper pass was requested.
- Every report opportunity has a post/current URL when available.
- Every report opportunity has a context-aware copy-ready comment.
- Every copy button copies the comment only and does not imply auto-posting.
- Comments are value-first, same-language, short, natural, and not direct ads.
- One or two small natural imperfections are allowed only when they help the comment sound human and do not reduce trust.
- Logs were updated.
- No auto outreach, auto DM, credential/secret collection, or access-control/CAPTCHA bypass occurred. (Collecting publicly-rendered contact details under the operator's command is permitted per the collection-consent rule above; the send/act side still needs human approval.)
