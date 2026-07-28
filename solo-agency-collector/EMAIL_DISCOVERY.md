# Finding a lead's email on Facebook — collector capability + how a playbook must use it

Status: capability shipped in extension build `0.1.51-profile-contacts`; the ladder itself
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
{ "profile_url": "...", "emails": ["info@bgv.com.vn"],
  "found_on": "current_page",
  "checked": ["current_page", "directory_contact_info"] }
```

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
5. **`directory_links`** — often carries the website when no address is published.

Sub-tabs are fetched same-origin from the operator's own logged-in session — the same
pages a click would open. Nothing hidden is expanded and the tab never navigates away.

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

**Retry once before concluding.** The same profile on the same build returned an address in one
run and nothing in the next, purely from render timing. A single empty result is not evidence.

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
4. **Only then escalate:** website found on the profile → its Contact/Team/About page and
   footer → off-platform search.

### Completion gate — when `email_not_found` is allowed

Do NOT mark `email_not_found` unless ALL of these hold:

- `fb.profile.contacts` ran against the resolved **profile** URL (not the reel/post), and
- its `checked` array contains `current_page` **and** `directory_contact_info`, and
- the data point shows `expanded_truncations` ≥ 0 with `opened_contact_tab` recorded
  (the collector attempted the un-truncation and the contact sub-tab), and
- the website ladder ran if `directory_links` produced one.

Anything less is "not looked for yet", not "not there".

### Records that must be REJECTED outright

Every collector data point carries these guards — a record with either flag set describes
someone other than the lead, so its email/phone must never be written to the dossier:

| Field | Meaning |
|---|---|
| `landed_on_self: true` | Facebook fell back to the **operator's own profile** (malformed profile URL or an unsupported tab slug). Its contact details are the operator's. |
| `url_drifted: true` | The page read is not the item that was requested (a reel player can advance to a recommended reel). |

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
