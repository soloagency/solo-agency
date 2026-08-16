# Zillow capabilities — the caller contract

Two Local Collector capabilities read **Zillow** the way `fb.profile.enrich` reads Facebook.
This file is for the agent that writes the CALLER (a playbook step, a harvest walker, a script):
what to submit, what comes back, what the caller owns, and what it must never do.
The machine-readable spec is `bridge-go/collector_capabilities.json` (`zillow.agents.list`,
`zillow.profile.enrich`, schemas `ZillowAgentCard`, `ZillowProfileEnrich`); this document
explains how to use them and does not repeat every field.

| Capability | Reads | Returns | Status |
|---|---|---|---|
| `zillow.agents.list` | ONE agent-directory page `zillow.com/professionals/real-estate-agent-reviews/<location>/?name=<kw>&page=<n>` | `items[]` = agent cards + page facts on the envelope | beta — live-verified 2026-08-16 (aven-ngo) |
| `zillow.profile.enrich` | ONE agent or team profile `zillow.com/profile/<screenName>` | one `ProfileEnrich`-shaped lead record + typed `zillow{}` block, `industry: "Real Estate"` | beta — live-verified 2026-08-16 (aven-ngo) |

Code: `chrome-extension/zillow_extract.js` (MAIN world, own file; injected by `background.js`
only for `zillow.*` jobs and dispatched through `window.__soloZillowRun` — the Facebook library
`gql_extract.js` is untouched). Offline test: `node solo-agency-collector/tests/test_zillow_extract.js`.
Both pages are Next.js: the extractor parses `<script id="__NEXT_DATA__">` (no CSS selectors) and
falls back to a thin DOM read only when that JSON is missing (`source: "dom"`).

---

## 1. Submitting a job

Same bridge, same job shape as every other capability. One source per job is the safe default
(two sources with the same url in one job are deduped; the bridge caps `max_sources` at 20).

```bash
curl -s -X POST http://127.0.0.1:17321/jobs/run_now -H "Content-Type: application/json" -d '{
  "run_id": "zillow_list_'$(date +%H%M%S)'", "client_slug": "<client_slug>", "run_now_ttl_minutes": 20,
  "pacing": {"scroll_steps": 0, "min_delay_seconds": 2, "max_delay_seconds": 3, "max_sources": 1},
  "sources": [{
    "name": "zillow agents los-angeles kim p1",
    "url": "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=kim&page=1",
    "source_type": "public", "platform": "zillow",
    "capability": "zillow.agents.list", "inputs": {}
  }]
}'
```

```bash
# profile — the url IS the input; max_posts / max_team_members are optional
"sources": [{"name": "zillow profile Pardee Properties",
             "url": "https://www.zillow.com/profile/Pardee%20Properties",
             "source_type": "public", "platform": "zillow",
             "capability": "zillow.profile.enrich", "inputs": {"max_posts": 5}}]
```

Rules the caller must respect:

- **The url is the request.** `zillow.agents.list` reads exactly the page it is given — keyword
  (`?name=`), location slug and `?page=N` are all yours. It never paginates and never clicks.
- **`platform: "zillow"`** in the source (otherwise the data point is labelled `web`; harmless,
  but you want to find these records later).
- Do not set `graphql_capture: false` on the job — capability dispatch is gated behind it.
- Pacing: keep Zillow jobs on ONE Chrome profile ≥ 15–30 s apart and the hourly volume modest.
  A Chrome profile that has never visited zillow.com should be warmed by the operator opening
  it once by hand. See §5 for what to do when Zillow pushes back.
- Jobs are serialized per client; the extension for `<client_slug>` must be loaded in that
  Chrome profile (a stale extension returns `records: null` for a capability it does not know —
  reload after every sync, and check the version in `chrome://extensions`).

## 2. Reading the result

`…/collector/inbox/YYYY-MM/<client_slug>/<run_id>/private_data_points.jsonl` — one line per
source, `collector_status.json.completed === true` when done. The typed output is
`records` (`{capability, available:true, status, count, items[], …meta}`); the lead record is
`records.items[0]` for a profile, `records.items[]` for a list.

`records.status` (envelope) is how the capability tells you what happened — read it before
`items`:

| status | meaning | caller action |
|---|---|---|
| `ok` | read the page, `count` items | use them |
| `empty` | page rendered, no agent cards (e.g. a location/keyword with no agents) | not an error; move on |
| `no_next_data` | the Next.js payload was missing; the DOM fallback ran (`source: "dom"`, fewer fields) | usable but degraded — treat as a layout-change alarm and tell the operator |
| `no_display_user` | profile page without a `displayUser` (not an agent profile / removed) | skip |
| `blocked` | Zillow bot check (`reason: "blocked_bot_check"`) | **STOP** — see §5 |
| `error` | the extractor threw (`error` says what) | log, retry once later, then report |

`available` is always `true` for these two capabilities, so `records` is never null because of
the page; **`records: null` means the extension did not run the capability at all** (stale
extension without `zillow_extract.js`, injection failure, or the 45 s dispatch timeout) —
that is a collector problem, not a page problem.

## 3. Walking a directory (the caller owns pagination)

The list envelope carries everything a walker needs:

```
page, page_count (+ page_count_source: "pager" | "estimated"), results_found,
has_more, next_page_url, non_profile_cards, query{url, location_text, name, applied_filters, region_id, region_name}
```

- Zillow serves **at most 25 directory pages per query** (~15 cards each), whatever
  `results_found` says (44,534 agents found for `los-angeles-ca` → still "Page 1 of 25").
  A big area is walked by **varying the keyword and/or location** (`?name=kim`, `?name=lee`,
  `venice-ca`, `90291`, …), not by paging deeper.
- Loop: submit page 1 → if `has_more` submit `next_page_url` (it is the same url with `page`
  advanced, keyword and filters kept) → stop when `has_more` is false or `page >= page_count`.
- Dedupe across pages/keywords by `encoded_zuid` (Zillow's stable id) with `profile_url` as the
  fallback key. The bridge's own `sourceUID` also collapses `zillow.com/profile/<x>` variants
  (case, `www.`, trailing slash) to one identity.
- `is_team: true` cards are teams/brokerages: their `zillow.profile.enrich` record lists every
  member in `zillow.team.members[]` (name, `screen_name`, `profile_url`, `encoded_zuid`,
  rating, reviews_count, is_top_agent) — each `profile_url` is a further enrich target.
- `non_profile_cards` counts the upsell cards Zillow mixes into the results ("Get help finding
  an agent"); they are already removed from `items`.

## 4. The profile record → a lead

`records.items[0]` uses the **same field names as `fb.profile.enrich` / `ProfileDossier`**, so
everything that reads an FB enrich record (harvest decide, the campaign UI drill-down,
`04_VERIFY_ENRICH`) reads this one unchanged. The mapping to a CRM contact is mechanical — no
judgment step is required:

| Lead field | Record field | Notes |
|---|---|---|
| name | `name` | display name (`"Tami Pardee"`) |
| email | `emails[0]` | present only when the agent published one on Zillow (`displayUser.email`; both live test profiles had one). Empty → run the usual off-platform email ladder (website / web.search); `found_on` is `null` then. |
| phone | `phones[0]` (labels in `zillow.phones{cell,business,brokerage}`) | deduped by digits |
| website | `website` (all links incl. socials in `websites[]`, typed in `zillow.socials{}`) | tracking params already stripped |
| location | `location[0]` (`"Venice, CA 90291"`); typed in `zillow.business_address{}`; coverage in `zillow.service_areas[]` | |
| industry | `industry` = `"Real Estate"` (always) | a fact of the source, not an inference |
| title / company | `category` (`"Broker"` or `"Real estate agent"`), `zillow.brokerage`, `work[]` | |
| zillow identity | `profile_url` (canonical `https://www.zillow.com/profile/<screenName>`), `zillow.encoded_zuid` | → `identities.socials.zillow` |
| socials | `zillow.socials{facebook,instagram,linkedin,x,tiktok,youtube}` | ready for `identities.socials.*` |
| dated proof-of-life | `posts[]` (`kind: "review"|"sale"`, `date`, `created_time`, `text`, `url`), typed in `zillow.recent_reviews[]` / `zillow.recent_sales[]` | newest first, `max_posts` (default 5); `timeline.posts_available` |
| trade facts | `about_lines[]` (one `Label: value` line each), `zillow.licenses[]`, `zillow.years_in_industry`, `zillow.specialties[]`, `zillow.sales{last_12_months,total,price_range,average_price}`, `zillow.rating{average,count}`, `zillow.listings{for_sale,for_rent}`, `zillow.is_top_agent`, `zillow.team{role,team_name,lead,members[],member_count}` | |

Example `contact add` shape (the operating brain still runs the CRM write, exactly as for FB):

```json
{"name":{"full":"Tyler Kunkle"},
 "identities":{"emails":["tyler@pardeeproperties.com"],"phones":["(310) 993-7333"],
               "socials":{"zillow":"https://www.zillow.com/profile/tykunkle","instagram":"https://www.instagram.com/tykunkle/"},
               "website":"https://pardeeproperties.com/team/tyler-kunkle/"},
 "tags":["zillow_directory"],
 "custom_fields":{"source":"zillow.profile.enrich","industry":"Real Estate","brokerage":"Pardee Properties","location":"Venice, CA 90291"}}
```

`bridge-go/leads.go` `classifyLeadURLFull` knows Zillow: `/profile/<x>` → `profile`/`zillow`
(accepted as `channels_found.profiles.zillow` by `enrich write`), the directory url → nothing
(a query page, never an identity), listing pages → `seed`. **This lands with the next bridge
deploy** (`deploy-soloagency.sh --collector-only`); a bridge built before 2026-08-16 rejects a
Zillow profile url under `channels_found.profiles` — put it in `identities.socials.zillow` via
`contact add` instead until then.

## 5. When Zillow pushes back — the blocked contract

Zillow runs PerimeterX. Its "Press & Hold to confirm you are a human" page is detected
(`#px-captcha`, title "Access to this page has been denied") and returned as
`status: "blocked"`, `reason: "blocked_bot_check"`, `available: true`, `count: 0`. Measured: a
fresh, non-human browser was blocked on its 2nd navigation; the operator's real Chrome was not
blocked across 5 live jobs.

The caller MUST:
1. stop submitting Zillow jobs for that Chrome profile / client immediately (a harvest walker
   should quarantine the collector, not retry the page);
2. hand the operator one `**[ACTION REQUIRED]**`: open `https://www.zillow.com/` in that Chrome
   profile, pass the Press & Hold check once by hand, reply `done`;
3. resume at a slower pace after `done`. Never automate the check, never spoof it, never route
   the page through another browser.

## 6. What these capabilities do NOT do (by design, as agreed 2026-08-16)

- No pagination, no keyword generation — caller-owned (§3).
- No CRM write — the record is parked in `private_data_points.jsonl` like every other data point;
  the operating brain maps it (§4).
- No harvest-daemon integration yet: `bridge-go/harvest_daemon.go` still hardcodes
  `fb.profile.friends` / `fb.profile.enrich` (`harvestStep`, ~line 298) and `platform: "facebook"`
  (~line 341); `harvest.go` names its seed leg `FriendsURL`/`EndCursor`. A `source_kind: "zillow"`
  branch there (leg = `zillow.agents.list` on `seed_url + ?page=N`, enrich = `zillow.profile.enrich`)
  is the natural Phase 2 — the state machine, ledger and UI are already capability-agnostic.
- Only the real-estate-agent directory. Lender / property-manager / photographer directories are
  other Zillow surfaces and are not read.
- `education[]`, `follower_count`, `verified`, `cta[]`, `discovered_tabs`, `graphql_*` are present
  but empty/null — they exist so FB readers do not break, they carry no Zillow meaning.

## 7. Maintenance

- Offline: `node solo-agency-collector/tests/test_zillow_extract.js` (fixtures = the live-observed
  `__NEXT_DATA__` shapes; 63 checks). Run it before every sync.
- Dev loop: `solo-agency-collector/sync-dev-extension.sh` → operator reloads the aven-ngo
  extension (version must change) → submit jobs (§1) → read `records` (§2). The repo's
  `chrome-extension/` is never loaded into Chrome directly.
- Catalog: the running bridge serves its embedded catalog until a rebuild; to expose the new
  entries live before that, copy `bridge-go/collector_capabilities.json` next to the running
  `collector_config.json` (read fresh, no restart) — operator's call, it is shared infrastructure.
- If Zillow renames JSON keys: records come back `status: "no_next_data"` / `source: "dom"` with
  fewer fields. Re-verify the paths listed in the catalog `_impl.method` (a `__NEXT_DATA__` dump
  from the operator's Chrome is enough) and update `zillow_extract.js` + the fixtures.
