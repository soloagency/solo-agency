# Leads From Zillow — the `zillow_harvest` campaign channel

Stage: `17`

## Load Rule

Load whenever a client has a campaign with `channel_strategy: zillow_harvest` — during Setup
Flow when one is created or edited, and in every scheduled run's step for collector-executed
channels (AUTOMATION_SCHEDULING §H). LOAD LEDGER applies. Read Stage 16 first: this channel is
the same machine with a different list source and no judgement step.

## What this channel is

Zillow's real-estate-agent directory, walked location × keyword × page by the bridge daemon,
each agent profile read once, and — because the record is deterministic and published by the
agent — written straight into the CRM by the daemon. **The agent has no judgement step here.**
Nothing is sent, nothing needs approval.

- **Inputs** (campaign config, edited in `/ui/{client}/campaign/{slug}` or `campaign update`):
  `zillow_locations[]` — one directory url per line
  (`https://www.zillow.com/professionals/real-estate-agent-reviews/<location>/?name=`; the
  `name`/`page` params are stripped on save, other Zillow filters in the url are kept);
  `zillow_keywords` — comma-separated (surnames work: "kim, nguyen, tran"), lower-cased and
  deduped on save; optional `harvest{}` overrides (`daily_budget`, `per_collector_budget`,
  `quiet_from/to`) — blank = system settings (`/ui/settings`, "Leads From Friends (harvest
  pacing)" applies to both channels).
- **The walk** (`harvest_daemon.go` + `harvest_zillow.go`): location[0] × keyword[0] page 1 →
  `zillow.agents.list` → cards deduped through the client-wide `harvest/seen_profiles.json`
  (uid = the profile url identity; a person already kept/rejected by ANY campaign of the
  client, Facebook or Zillow, is skipped) → new cards queued → each queued profile read once
  with `zillow.profile.enrich` (20–40 s apart, rotating collector accounts, per-account caps,
  quiet hours — Stage 16 rules verbatim) → next page → when a page comes back **empty**, or the
  envelope says `has_more: false`, or `page >= page_count` (Zillow serves at most 25 pages per
  query — coverage comes from more keywords/locations, not depth), the query ends → next
  keyword at page 1 → after the last keyword, next location → after the last location the
  campaign is exhausted. Team cards are read too; their `zillow.team.members[]` become further
  targets.
- **The CRM step is the daemon's** (operator ruling 2026-08-16): a profile with an **email or a
  phone** → `contact add` (match-or-create by identity: `identities.emails/phones/socials.zillow/
  website`, `tags: ["zillow_directory","zillow_harvest"]`, `custom_fields.source/industry/
  brokerage/location/zillow_zuid/title`) → registry `kept` with the `lead_id`; a profile with
  **neither** → registry `rejected` ("no email or phone published on Zillow") — skipped, never
  read again. `industry` is always `"Real Estate"` (a fact of the source).
- **Bot check ("Press & Hold")**: the collector handles it end to end (tab to front, chime, wait
  up to 6 h, heartbeat every 2 min, continue when the operator passes it). The daemon treats a
  job whose latest source_status is `waiting_for_human_captcha` as IN PROGRESS — never stale,
  never a failure, never a quarantine. Only a FINAL blocked record (ceiling passed,
  `records.status: "blocked"`) benches that account for the day and re-schedules the same page
  on another account; that bench is account-side and is NOT lifted by wake amnesty. The
  per-campaign `zillow.blocked[box]` mark is day-scoped and is cleared the moment that
  account reads a Zillow page cleanly again (the operator passed the check) — the daemon
  resumes on its own. A directory page that keeps failing across accounts (4 consecutive
  failed legs) is recorded in `zillow.errors[]` and SKIPPED, so one broken page never stalls
  the walk; a degraded read (`no_next_data`, 0 cards) is retried, never taken as "end of query".

## Setup

1. Create the campaign with `channel_strategy: zillow_harvest`, paste the location urls (one per
   line) and the keywords, save. Created **paused** (same reason as Stage 16: it touches a live
   site on its own) — press **Start harvest** when the inputs are final. Pause/Start keeps the
   cursor (`progress.json` → `zillow{loc_idx, kw_idx, page, queries_done, blocked}`).
2. Warm each collector Chrome profile once by hand on `https://www.zillow.com/` before the first
   run (ZILLOW_CAPABILITIES §1) — a profile that has never visited Zillow is blocked on its
   second navigation.

## Every scheduled run — what the agent does (small)

There is no judgement pass. The run:

1. `tool crm-store --client-dir {outreach} harvest status --campaign X` → one progress line in
   the run reply from its `zillow{}` block + totals: location/keyword/page (`loc_idx`/`kw_idx`/
   `page` against `config.zillow_locations`/`zillow_keywords`), `queries_done`, `cards_seen`,
   read today / budget (`day_enriched`), `totals.kept` (added to CRM) / `totals.rejected`
   (skipped, no email/phone), `zillow.blocked` (accounts that hit the final bot-check block
   today) and `zillow.errors[]` (query pages skipped after repeated failures). Collector
   liveness is on the campaign page (bridge-side), not in the CLI output.
2. If `zillow.blocked` names an account: ONE `**[ACTION REQUIRED]**` — open `https://www.zillow.com/`
   in that client's Chrome profile and pass the check once; the daemon resumes on its own.
3. Never enqueue `zillow.agents.list` / `zillow.profile.enrich` jobs yourself for a harvest
   campaign (Stage 16 rule); never "help" a paused walk — report `harvest status`.
4. Kept contacts are already in the CRM (`source: zillow_harvest`); whether they enter an
   email/DM campaign is the normal, separate decision.

## Storage (Stage 7 addendum)

Same files as Stage 16 (`harvest/seen_profiles.json`, `harvest/{campaign}/progress.json` with a
`zillow{}` cursor block, `harvest/{campaign}/enriched/{uid_hash}.json` for the drill-down,
`collector/harvest_ledger.json`), plus campaign config keys `zillow_locations[]`,
`zillow_keywords[]`. Collector output lands under the owning client's inbox tree
(`collector/inbox/YYYY-MM/{owner}/harvest/{run_id}/`) regardless of which extension read it.
