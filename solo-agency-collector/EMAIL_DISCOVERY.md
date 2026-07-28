# Finding a lead's email on Facebook — collector capability + how a playbook must use it

Status: capability shipped in extension build `0.1.58`; the ladder itself
was measured on real profiles (evidence below). Read this before writing any enrichment
prose about "we couldn't find an email".

## The one rule

**A business almost always publishes its address on its OWN profile.** Going to the
website or an off-platform search *before* the profile has actually been dug through is
the #1 reason enrichment reports `email_not_found` for a lead whose address was two clicks
away. Exhaust the profile first; escalate only after the ladder has genuinely run.

## Capability: `fb.profile.contacts`

One job digs the whole profile and returns any published address.

```json
{
  "name": "email dig <lead>",
  "url": "https://www.facebook.com/<vanity-or-profile.php?id=NNN>",
  "platform": "facebook",
  "capability": "fb.profile.contacts",
  "inputs": { "stop_at_first": true }
}
```

| Input | Meaning |
|---|---|
| `profile_url` | optional; defaults to the page the job opened. Vanity (`/bgvinvest`) or numeric (`profile.php?id=61581802543283`). A URL that already carries a tab (`…&sk=reels_tab`, `/about`) is trimmed back to the profile base automatically. |
| `stop_at_first` | optional, default `true` — stop at the first surface that yields an address. `false` collects from every surface. |

Output — `records.items[0]` is a `ContactRecord`:

```json
{ "profile_url": "https://www.facebook.com/bgvinvest",
  "emails": ["info@bgv.com.vn"],
  "websites": ["https://bgv.com.vn"],
  "found_on": "current_page",
  "checked": ["current_page", "about", "directory_contact_info"] }
```

| Output | Meaning |
|---|---|
| `profile_url` | the profile BASE the job resolved — **not** a sub-tab URL, even when the answer came from one |
| `emails` | published addresses, junk domains filtered |
| `websites` | outbound links found on the profile (usually from `directory_links`). This IS the website hop's input — do not re-scan the profile for it. |
| `found_on` | the surface that yielded the address, or `null` |
| `checked` | the surfaces that actually RENDERED, in visit order. Includes `about` when the ladder had to enter the About section first — which is the normal case for a plain profile URL. |

### What it actually does (the ladder)

**The sub-tabs are CLICKED, never fetched.** A same-origin `fetch()` of
`/directory_contact_info` returns the SPA shell without the contact block: across 68 profiles
every address came from the rendered page and **zero** came from a fetched tab — including one
whose email sat in Contact info the whole time. Worse, counting those fetches as "checked"
produced an audit trail that claimed the ladder had run when it had not. The capability now
enters About and clicks each sub-tab (matched by the slug in its `href`, since the visible
label differs between Pages, personal profiles and locales), waiting for the text to stop
changing before reading — a fixed pause was not enough, and that race, not scrolling, is what
made discovery flaky.


1. **The rendered profile page** — including text the collector just un-truncated, because
   Facebook hides the rest of a long bio behind **"See more"** and that text is not in the
   DOM until it is expanded.
2. **`directory_contact_info`** — vanity: `/<name>/directory_contact_info`; numeric:
   `profile.php?id=<id>&sk=directory_contact_info`.
3. **`directory_intro`**
4. **`directory_basic_info`**
5. **`directory_links`** — often carries the website when no address is published; what it
   yields comes back in `websites`.

**Known limit — "See more" is expanded once, on the entry page only.** The un-truncation runs
in the generic capture step before this capability is even injected, and is not re-run on the
sub-tabs the ladder then clicks into. An address truncated *inside* a sub-tab can be missed, so
a sub-tab miss is weaker evidence than a main-page miss. It has not bitten a real lead yet
(sub-tab contact blocks are short).

### Measured on a 71-lead run (2026-07-28)

| Outcome | Leads | Share |
|---|---|---|
| Email published on Facebook | 30 | 42% |
| No email, but a website/channel is listed | 14 | 20% |
| Nothing on Facebook → external sources | 27 | 38% |

**82% of the emails were only reachable after expanding "See more"** — a profile whose bio is
truncated reports no email until the toggle is clicked. All of them were found on the rendered
profile page; the sub-tabs added 2 more. Two consecutive runs on the remaining leads produced no
further addresses, so 38% is a converged figure, not an untried one.

**"No Contact info tab" is a valid conclusion.** Most personal profiles simply do not offer that
sub-tab — verified by hand on 13 profiles, of which 12 genuinely had none. `checked` lists only
the surfaces that really rendered, so a missing tab is absence of the tab, not a missed click.

### Earlier per-profile detail (2026-07-28)

| Profile | main page | contact_info | intro | basic_info | links |
|---|---|---|---|---|---|
| `bgvinvest` (`info@bgv.com.vn`) | ✅ | ✅ | ✅ | ✅ | ✅ |
| `Khanhngo.us` (`khanh8601@gmail.com`) | ✅ *(only after "See more" was expanded)* | ✅ | — | — | — |

Before "See more" expansion shipped, `Khanhngo.us` returned `emails: []` on the main page,
on `/about`, AND on `/about_contact_and_basic_info` — the address was on screen the whole
time. That is why "we opened the About tab" is not evidence that no email exists.

## How the playbook must use it

1. **Resolve the profile first.** A reel/post seed is not a profile. Reel → owner via the
   collector (`sk=reels_tab` link); post/permalink/story → owner is in the URL itself.
2. **Run `fb.profile.contacts` on that profile.** One job. Do not hand-write a tab crawl.
3. **Read `records.items[0].emails`.** Empty → check `checked` before drawing a conclusion.
4. **Copy the record into the dossier as `email_discovery`** — `{profile_url, emails, websites,
   found_on, checked}`, verbatim from the collector. The store's `mark_email_not_found` gate
   reads its `checked`; a dossier without it is refused (see below). **Also put any address into
   `identity.channels_found.emails`** — that is the canonical field that creates the identity.
   (`email_discovery.emails` is accepted as a source too, so a verbatim copy no longer loses the
   address, but `channels_found` is what the rest of the pipeline reads.)
5. **Only then escalate:** `websites` → that site's Contact/Team/About page and footer →
   off-platform search.

### Completion gate — when `email_not_found` is allowed

Enforced in code (`enrich write`). Do NOT mark `email_not_found` unless ALL of these hold:

- `fb.profile.contacts` ran against the resolved **profile** URL (not the reel/post), and
- the dossier carries `email_discovery.checked` from that run, and it shows the ladder got
  **past `current_page`** — either a `directory_*` sub-tab rendered, or `about` is present and
  the profile offered no sub-tab to click (the normal case for a personal profile), and
- the website ladder ran if `websites` produced one.

Anything less is "not looked for yet", not "not there".

**Gate on `checked`, never on `opened_contact_tab`.** `opened_contact_tab` and
`expanded_truncations` are generic pipeline flags on the data point, not this capability's audit
trail. `opened_contact_tab` only goes true when the JOB's url was *already* an about/directory
url, so on the recommended job shape (a plain profile URL) it stays **false even on a run that
clicked every sub-tab and found the address**. And `expanded_truncations` is a count, so "≥ 0"
is true of every run ever. Neither can carry a gate.

### Records that must be REJECTED outright

Every collector data point carries these guards — a record with either flag set describes
someone other than the lead, so its email/phone must never be written to the dossier:

| Field | Meaning |
|---|---|
| `landed_on_self: true` | Facebook fell back to the **operator's own profile** (malformed profile URL or an unsupported tab slug). Its contact details are the operator's. Detected on the job's ENTRY page, before the ladder's own clicks. |
| `url_drifted: true` | The page read is not the item that was requested. Only fires for reel/video jobs (a reel player can advance to a recommended reel) — it is inert on a profile job, so it is not a contacts-run health check. |

Known trap: for a numeric profile, `profile.php?id=<id>/about` and
`profile.php?id=<id>&sk=about_contact_and_basic_info` both land on the operator's own page.
Use the `directory_*` slugs above (or just let `fb.profile.contacts` build the URLs).

## Related: the passive `contacts` field

Every collection (any capability, any page) also returns an additive
`contacts: { emails, phones }` parsed from the page's visible text plus `mailto:`/`tel:`
anchors. It is a free by-product — use it when it happens to be populated, but it only sees
the page that was opened. `fb.profile.contacts` is the deliberate dig; `contacts` is not a
substitute for it.

Phone note: bare digit runs longer than 11 characters with no `+` and no separators are
Facebook object ids, not phone numbers, and are dropped (they used to surface as "phones").
