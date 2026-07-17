# Import A Lead List

Stage: `03`

## Load Rule

Load this stage whenever you import a lead list — a CSV, TXT, or XLSX file — into a client's contact set: first-list import during setup (Stage 1, Step 5), any add-a-list request, or a re-import/refresh of an existing list. It is loaded together with Stage 7 (`playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`), which owns the on-disk schemas this stage writes to. Every load needs a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md` (read to the last line; compare `playbooks/LOAD_MANIFEST.md` when present).

Import creates the contact set. It does **not** verify-enrich for send, does **not** guess or generate email addresses, does **not** draft, and does **not** send. Enrichment and guessing are Stage 4; drafting is Stage 6; sending is Stage 8. In Setup Flow, this stage imports and dedupes only (the Setup Flow hard stop still applies).

## Hard Gates For This Stage

- **Inspect before you import.** Always run `import_leads.py inspect` first, show the human the proposed column mapping and the sample rows in an `**[ACTION REQUIRED]**` block, and import only with a mapping the human has confirmed or corrected. Never import a file the human has not seen mapped.
- **All CRM mutations go through `crm_store.py`.** `import_leads.py` creates contacts by calling `crm_store` (`add_contact`) — that is the one sanctioned write path. Never hand-write `crm/contacts/*.json` or `crm/contact_identities.jsonl`. Direct writes to any `crm/` collection are a critical violation (inherited "no one-off scripts" rule).
- **Email is not required.** A row with only a name plus a phone or a social URL is a valid contact (DESIGN §7.1). Do not drop rows for lacking an email.
- **Suppression is checked at import against ALL identities.** The importer checks every email, phone, social URL, and the email's domain against both suppression tiers (global + client). A matched identity is recorded `suppressed` and no contact is created for it. Never bypass this.
- **Dedupe is exact-identity, not fuzzy.** A row whose normalized email / phone (E.164) / social URL already exists as a contact is recorded `matched`, not created. It reuses the existing `lead_id`.
- **The MX check marks, it does not block.** A failed MX lookup marks that email's status `email_not_found`; the contact is still imported. MX proves the domain can receive mail, never that the mailbox exists (DESIGN §9.6).
- **No guessed addresses here.** Import only records addresses *found in the file* (`source: import`). It never guesses or patterns an address — guessed/unverified addresses are a Stage 4 concern with its own third-party verification and send-time gate.
- **Import is idempotent.** Re-running the same file with the same mapping under the same list slug is a no-op (matched by an idempotency key over file content + mapping). Do not "force" a duplicate import to make counts look busier.
- Load Stage 7 IN FULL (LOAD LEDGER printed with `Verdict: PASS`, matching `LOAD_MANIFEST.md` when present) before writing any list artifact. A partial read = NOT loaded.
- Every human step in this stage — the mapping confirmation and any blocker — uses the `**[ACTION REQUIRED]**` block from `OUTREACHCRM_PLAYBOOK.md`: one purpose, one exact next step, one command or path. When the import is clean, say `No action required right now.`

## Source Preservation Rule

This file is the detailed source material for list import. Do not summarize away the inspect-then-confirm flow, the exact tool commands and flags, the mapping synonyms, the dedupe/suppression/idempotency rules, the record schemas, the completion gates, or the warnings. A downstream agent may shorten its human-facing reply, but it must still obey the full requirements here. If you cannot fit a schema, load the schema (Stage 7 §5) — do not reconstruct it from memory.

## Phase Status — These Tools Exist

`tools/import_leads.py`, `tools/crm_store.py`, and `tools/email_verify.py` ship in Phase 1 and are present in this checkout. The DESIGN §22 R2 `tool_not_built` honest-blocker path therefore does **not** apply to them: importing a list is a real, required step here, not something to record as `skipped: tool_not_built`. (Stages 4 verify/enrich, 5 campaign, 6 email writing, and 10 follow-up that this stage hands off to are now **built** (Phase 2, 2A–2D); only Stages 12/15 remain `status: planned` — Phase 3. Regardless, do not attempt enrichment or drafting from this stage on their behalf; import, dedupe, suppression-check, and MX-mark only, then hand off.)

---

## 1. The Inspect → Confirm → Import Flow

Import is a two-command flow with a human confirmation in the middle. Never skip the inspect step.

### 1.1 Step 1 — Inspect

Run `inspect` to read the file's headers, get a proposed column mapping, and see sample rows. `inspect` reads the file only; it writes nothing and needs no client directory.

```bash
python3 tools/import_leads.py inspect --file /path/to/list.csv
```

Optional: `--rows N` to change how many sample rows are printed (default 5).

`inspect` prints JSON:

```json
{
  "headers": ["Email", "Full Name", "Office Name", "Cell/Office Phone", "Website", "City", "State"],
  "proposed_mapping": {"email": "Email", "full_name": "Full Name", "company": "Office Name", "website": "Website", "city": "City", "state": "State"},
  "sample_rows": [ { "...": "..." } ],
  "total_rows": 412,
  "note": "Confirm/adjust the mapping, then run: import_leads.py import --mapping '<json>'"
}
```

The mapping is `{canonical_field: source_column_header}`. Canonical fields the importer understands: `email`, `full_name`, `first_name`, `last_name`, `company`, `phone`, `website`, `city`, `state`, `facebook`, `linkedin`, `instagram`. The value is the exact column header in the file.

Auto-mapping is by header synonym (case-insensitive, exact-header match — not substring). The recognized synonyms are:

| Canonical field | Header synonyms recognized |
|---|---|
| `email` | email, e-mail, email address, mail |
| `full_name` | full name, name, contact name, fullname |
| `first_name` | first name, first, firstname |
| `last_name` | last name, last, lastname, surname |
| `company` | company, office name, brokerage, organization, org, business |
| `phone` | cell phone, mobile, cell, phone, phone number, office phone |
| `website` | website, url, web, site |
| `city` | city, office city |
| `state` | state, office state |
| `facebook` | facebook, fb |
| `linkedin` | linkedin |
| `instagram` | instagram, ig |

A header the synonym table does not recognize is left unmapped and must be added by hand in the confirmed mapping (see the realtor example in §5). This is exactly why the human confirms the mapping before import.

### 1.2 Step 2 — Confirm the mapping with the human

Show the proposed mapping and the sample rows and ask the human to confirm or correct. This is the required human gate:

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**I need you to:** confirm the column mapping below, or correct any column that is wrong or missing. Any columns you have (name, email, company, phone, profile URL, city, state) are fine; email is not required.
**Proposed mapping:** {the proposed_mapping JSON}
**Sample rows:** {2–3 sample_rows shown readably}
**Reply with:** `mapping looks right`, or the corrected mapping (e.g. `map phone to "Cell/Office Phone"`)
**Why:** The importer builds every contact from these columns. A wrong or missing mapping loses data (for example, an unrecognized phone column would import contacts with no phone).
```

Hold the confirmed mapping for Step 3. Do not import before the human answers.

### 1.3 Step 3 — Import

Run `import` with the confirmed mapping. Import needs the client workspace directory and a list slug.

```bash
python3 tools/import_leads.py import \
  --client-dir daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outreach \
  --file /path/to/list.csv \
  --list-slug {list-slug} \
  --mapping '{"email":"Email","full_name":"Full Name","company":"Office Name","phone":"Cell/Office Phone","website":"Website","city":"City","state":"State"}'
```

Flags:

- `--client-dir` — the client workspace root: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outreach/`. The importer writes the list under this directory and routes contact creation to that client's `crm/` through `crm_store`.
- `--file` — the source file (csv/txt/tsv/xlsx).
- `--list-slug` — a slug for this list (lowercase, hyphens, no punctuation), e.g. `austin-realtors-2026-07`. Output lands in `lists/{list-slug}/`.
- `--mapping` — the confirmed mapping JSON. If omitted, the importer falls back to the auto-proposed mapping; only omit it when `inspect` already mapped every column you need and the human confirmed that.
- `--no-mx-check` — skip the MX check (see §7). Off by default; the MX check runs unless you pass this.

Import prints the resulting manifest (or a `skipped` result on an idempotent no-op — see §8).

---

## 2. Supported Formats

`source_format ∈ csv | txt | xlsx` (DESIGN §5.1; the importer also accepts `.tsv`). Stdlib-only readers:

- **CSV / TSV** — delimiter is sniffed (`,` `\t` `;`); the first row is the header. UTF-8 with BOM is handled.
- **TXT** — if the file has no comma or tab, it is treated as one email per line (header-less, single `email` column). Otherwise it is parsed like a delimited file.
- **XLSX** — the first worksheet, first row = headers, read with a minimal built-in zip/XML reader (no third-party library needed).

Any other extension is rejected. Convert to csv/txt/xlsx first and re-inspect.

---

## 3. Email-Optional Model

A contact does not need an email. Import keeps a row when it has **any** of: email, phone, a social URL, or a full name. A row with none of those (no identity and no name) is recorded `skipped_invalid`.

- With an email → `channels.email.status = usable` and the address is stored `source: import`, `status: unverified` (or `email_not_found` if the MX check fails — §7).
- Without an email → the contact still imports with `channels.email.status = needs_data`; it flows to assisted channels later (Stage 10) if consent exists. This is the name + profile-URL case (e.g. a realtor with a Facebook page but no listed email).

Import stores what it reads: `name`, `identities.emails/phones/socials/website`, and `company`/`city`/`state` into `contact.custom_fields`. It does not create Account records or resolve accounts — company clustering and account linkage are later-stage CRM work (Stage 13). Imported phones are stored E.164-normalized with `type: cell`.

---

## 4. Dedupe (find_by_identity — exact match, not fuzzy)

For each row, `crm_store.add_contact` checks every identity against the client's unique reverse index (`find_by_identity`) before creating anything:

- **Email** — lowercased and trimmed.
- **Phone** — normalized to E.164.
- **Social** — canonical profile URL.

If any of those already maps to an existing contact, the row is recorded `matched` and reuses that contact's `lead_id` (following merge chains via `resolve()`); no new contact is created. Otherwise a new `lead_id` (a `c_`+ULID, minted at import because email may be absent) is created and its identities are registered in the reverse index. Website is stored but is **not** a dedupe key.

Dedupe is per client and spans all of that client's lists — importing the same person from a second list matches the existing contact. This is distinct from idempotency (§8): dedupe works row-by-row on identity; idempotency short-circuits a whole re-import of the same file+mapping.

---

## 5. The Realtor-CSV Example (worked end to end)

The realtor list has these columns: `Email`, `Full Name`, `Office Name`, `Cell/Office Phone`, `Website`, `City`, `State`.

**Inspect:**

```bash
python3 tools/import_leads.py inspect --file ~/lists/al-realtors.csv
```

Auto-mapping recognizes `Email → email`, `Full Name → full_name`, `Office Name → company`, `Website → website`, `City → city`, `State → state`. It does **not** auto-map the phone: the header `Cell/Office Phone` is not one of the recognized phone synonyms (which include `cell phone` and `office phone`, but not the combined literal `cell/office phone`), so `phone` is left out of the proposed mapping.

**Confirm:** show the human the proposed mapping and note that the phone column needs to be added, then take their confirmation. Corrected mapping:

```json
{"email":"Email","full_name":"Full Name","company":"Office Name","phone":"Cell/Office Phone","website":"Website","city":"City","state":"State"}
```

**Import:**

```bash
python3 tools/import_leads.py import \
  --client-dir daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outreach \
  --file ~/lists/al-realtors.csv \
  --list-slug al-realtors \
  --mapping '{"email":"Email","full_name":"Full Name","company":"Office Name","phone":"Cell/Office Phone","website":"Website","city":"City","state":"State"}'
```

Result: each realtor becomes a contact with the email (if present) `source: import`, the phone E.164-normalized, `Office Name`/`City`/`State` in `custom_fields`, and the website stored. A realtor with a blank email still imports (email-optional). Any realtor already suppressed or already a contact is `suppressed` / `matched` rather than created.

---

## 6. Suppression At Import (all identities, both tiers)

Before creating a contact, the importer checks the row against suppression. It matches on the row's email, phone, and social URLs, **plus the email's domain**, against:

- **Agency tier** — `daily-content-pipeline/suppression/global_suppression.jsonl`.
- **Client tier** — `crm/suppression.jsonl`.

Any hit records the row `suppressed` with the suppression `reason` (e.g. `unsubscribe`, `hard_bounce`, `guessed_domain_kill`); no contact is created and no email is ever queued for that identity. A `domain`-tier suppression (for example a killed guessed domain, or a do-not-contact domain from the profile suppression policy) blocks every address at that host. This is the opt-out reach guarantee (DESIGN §16): suppression is honored at import, not only at send.

You can spot-check an address before or after import:

```bash
python3 tools/crm_store.py --client-dir <DIR> suppress check --email name@example.com --phone +15125551234
```

---

## 7. MX Check (marks status, never blocks)

Unless `--no-mx-check` is passed, each imported email is checked with `email_verify` (syntax + MX only, no SMTP probe). Outcome:

- **MX resolves** → the email keeps `status: unverified` (the domain can receive mail; the mailbox is still unproven).
- **MX fails / domain does not resolve** → the email's `status` is set to `email_not_found`. The contact is **still imported** — the address is kept and flagged, not dropped.

MX is deliberately weak: catch-all domains accept any recipient, so MX success is not deliverability proof and MX failure is the only strong signal here (it will hard-bounce). Real mailbox verification for guessed/pattern addresses is a Stage 4 concern (a third-party verification API), enforced at send time in `gmail_client.py send` — never invented in this stage.

You can check a single address directly:

```bash
python3 tools/email_verify.py check --email name@example.com
```

Status is one of `syntax_invalid | mx_ok | mx_fail`.

---

## 8. Idempotency (same file + mapping = no-op)

Every import computes an idempotency key = sha256 over (file content + mapping). It is stored in the list manifest. Re-running `import` on the same list slug with the same file and the same mapping matches that key and returns immediately without re-creating anything:

```json
{"skipped": true, "reason": "already imported (idempotency_key match)", "manifest": { "...": "..." }}
```

Consequences to keep straight:

- **Changing the mapping** (or the file content) changes the key, so the import runs again and `leads.jsonl` is rewritten for that list slug (a fresh `import_log.md` row is appended).
- **Same file under a different `--list-slug`** is a different manifest, so idempotency does not short-circuit it — but per-row dedupe (§4) will mark every already-known identity `matched`, so you do not get duplicate contacts.

Do not delete a manifest or salt a slug just to re-run an import; a genuinely new file gets a new slug, a re-run of the same file is meant to be a no-op.

---

## 9. What Gets Written

All list artifacts land under `lists/{list-slug}/` in the client workspace. These are list-scope config files (outside `crm/`), written by `import_leads.py` directly — they were never `crm_store` collections (Stage 7 §5). The contacts themselves are written to `crm/` **only** through `crm_store`.

- **`lists/{list-slug}/leads.jsonl`** — append-only-per-run audit, one row per source row with its outcome, so the import is re-runnable and reviewable:
  ```json
  {"seq":1,"ts":"","raw":{"Email":"...","Full Name":"..."},"normalized":{"email":"","full_name":"","company":""},
   "outcome":"created|matched|suppressed|skipped_invalid","lead_id":"c_...","reason":""}
  ```
  `outcome ∈ created | matched | suppressed | skipped_invalid`. `lead_id` is set for `created` and `matched`, null for `suppressed`/`skipped_invalid`; `reason` carries the suppression reason or the skip reason.

- **`lists/{list-slug}/list_manifest.json`** — the counts and provenance for this list:
  ```json
  {"schema_version":1,"list_slug":"","source_file":"","source_format":"csv|txt|xlsx","imported_at":"",
   "idempotency_key":"","column_mapping":{},"row_count":0,"contacts_created":0,
   "contacts_matched_existing":0,"suppressed_at_import":0,"rows_skipped":0,"notes":""}
  ```

- **`lists/{list-slug}/import_log.md`** — a human-readable table, one row per import run: date, source, rows in, created, matched, suppressed, skipped, blocker.

Each `created` contact also gets an `imported` activity appended to `crm/activities/YYYY-MM/activities.jsonl` (through `crm_store`, `by: agent`), so the contact timeline shows where it came from.

Reconciliation check: `row_count == contacts_created + contacts_matched_existing + suppressed_at_import + rows_skipped`. Every source row lands in exactly one bucket. If they do not sum, treat it as a blocker and investigate before relying on the list.

---

## 10. The Import Summary (only an ACTION REQUIRED on a blocker)

After a clean import, report the counts and end with `No action required right now.` — the human does not need to do anything for a normal import.

Raise a single `**[ACTION REQUIRED]**` block only when there is a real blocker, for example:

- **Could not infer a column mapping** and none was passed → the tool exits asking for `--mapping`. Action: pass the confirmed mapping.
- **Unsupported file type** → the tool rejects it. Action: convert to csv/txt/xlsx and re-inspect.
- **`contacts_created` is 0 while `row_count` is greater than 0** (everything matched, suppressed, or skipped) → surface it: the list added no new contacts, which is usually a wrong-list or already-imported signal worth a human look.
- **A large `rows_skipped`** (many rows with no identity and no name) → likely a mapping problem (e.g. the name column was not mapped); surface it and re-inspect.

Example clean summary:

```text
Imported al-realtors: 412 rows → 380 created, 18 matched (already known), 9 suppressed, 5 skipped (no identity).
List: daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outreach/lists/al-realtors/
No action required right now.
```

Example blocker:

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**I need you to:** re-check the mapping — 380 of 412 rows were skipped because no name or identity column mapped.
**Reply with:** the corrected mapping (map the name/email/phone column), or `re-inspect`
**Why:** With the name and identity columns unmapped, the importer sees empty rows and creates no usable contacts.
```

Do not end an import handoff with only a count or a bare path when a blocker exists; name the one next step.

---

## 11. Completion Gates For This Stage

Before calling a list import complete:

- Stage 7 was loaded IN FULL with a LOAD LEDGER `Verdict: PASS` before any list artifact was written.
- `inspect` was run and the human confirmed or corrected the mapping before `import` ran.
- The import ran through `python3 tools/import_leads.py import` (contacts created via `crm_store`); no `crm/contacts/*.json` or `crm/contact_identities.jsonl` was hand-written.
- `lists/{list-slug}/` contains all three artifacts: `leads.jsonl`, `list_manifest.json`, `import_log.md`.
- The manifest counts reconcile: `row_count == created + matched + suppressed + skipped`.
- Suppression was checked at import against all identities and both tiers (it is, by the tool) and no suppressed identity became a contact.
- No enrichment, guessing, drafting, or sending was performed in this stage — those are Stages 4/6/8.
- A blocker, if any, was surfaced with one `**[ACTION REQUIRED]**` block; otherwise the reply ends with `No action required right now.`

---

When any instruction here conflicts with `docs/DESIGN.md`, `docs/DESIGN.md` wins — follow it and report the gap.
