# Client Setup And Profile

Stage: `01`

## Load Rule

Load this stage during first-client setup, the add-client flow, setup repair, and the Automation Flow first agency run. It must be loaded together with Stage 0 (`playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`) before the first setup question, and every load needs a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md` (read to the last line; compare `playbooks/LOAD_MANIFEST.md` when present).

In Setup Flow, the send/enrich/campaign parts of the workflow this stage points at are superseded by the Setup Flow hard stop. Setup Flow configures OutreachCRM so a client-specific automation task can send later. It never sends an email, never enriches for send, and never runs a campaign.

The Client Intelligence Profile that this stage builds is written using the canonical schema in Stage 7 (`playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`); this stage owns the interview that fills it, not the schema itself.

## Hard Gates For This Stage

- The first setup question asks only one open thing: what the client sells and to whom (a product/service description, a website or profile URL, and target location only if it matters). Do not ask a dozen questions.
- Do not ask the human to define ICP, pain points, value proposition, brand voice, offer, or audience by hand. The agent infers these from the one opening answer and shows them for correction.
- When the sibling Solo Agency content-pipeline Client Intelligence Profile already exists for this client, follow the Solo Agency Profile Bootstrap rule (below): pre-fill from it, open with the inference block instead of the first setup question, and let the human confirm instead of re-declaring anything.
- Show the inference before asking the next question. The human sees the agent's reasoning evolve after every answer.
- Explain any marketing/deliverability/compliance term in plain language when asking the human for input.
- Setup Flow NEVER sends an email, NEVER enriches a contact for send, NEVER runs a campaign, and NEVER previews-then-sends. It only creates config and the client-specific automation task.
- Terminal state for Setup Flow is `ready_for_automation_first_run` (or `ready_for_next_automation_run` for an already-live client), never `first_send_completed`.
- If the human asks to send, enrich-and-send, run a campaign, "email them now", or "just send the first batch" while this stage is being used for Setup Flow, do not send. Refuse with the exact wording in the Setup Flow Send Refusal section, finish or resync the client-specific automation task, and tell the human the exact task name to run.
- The first send, enrichment pass, and approval batch happen in Automation Flow, driven by a client-specific task whose name begins with the client name, for example `AvenNgo - Buyer Leads Intro Daily Run`. The first run is simply that task's first execution, not a separate task.
- Nothing leaves the system without an explicit chat `approve`. Even in Automation Flow the send is gated by the Preview & chat-approval step (Stage 5 / Stage 8). Setup Flow does not reach that gate at all.
- Every human step in this stage — every question, approval request, API-key request, one-line command, and native automation task edit — uses the `**[ACTION REQUIRED]**` block from `OUTREACHCRM_PLAYBOOK.md`. When nothing is needed, end with next-action guidance per the Next-Action Guidance Rule.
- Load the referenced stages (2 sendbox, 3 import, 5 campaign, 7 storage schema, 12 analytics) with their own LOAD LEDGER before writing the config those stages own. Do not hand-write sendbox, list, campaign, pipeline, or analytics files from prose.
- Configure the recurring schedule and the client-specific automation task once the profile and first campaign goal are known. Then offer the agency-wide maintenance task `OutreachCRM - GitHub Update Watch` as a separate update-watch automation.

## Campaign Quick Start (the default when Solo Agency setup exists)

When the client already completed Solo Agency content setup (the Stage-1 bootstrap source exists), campaign setup collapses to THREE human questions, asked in the SAME setup session. The system is complex so the human does not have to be: never ask about anything below that can be defaulted or reused.

The only three asks:

1. **The lead list** — a CSV/file/pasted rows, or an approved handoff package from the content automation. The list is USER-CURATED: import it whole per the User-Curated List Rule (Stage 3) — the agent never judges fit.
2. **Sendbox App Password + sending identity** — ONE combined message (full create steps per Stage 2 §2.1): the sending Gmail + App Password via env var, plus from-name/title/reply-to/physical mailing address for the CAN-SPAM footer (legally required, cannot be inferred). Asked ONCE per client; later campaigns reuse the sendbox and identity with no question. Skip this ask entirely when a healthy sendbox already exists.
3. **Goal + companion URL confirmation** — ONE compact block: the inferred goal (goal_type/objective/CTA), the companion URL (ask whether there is one; none is fine), the proposed failure policy, a SHORT summarized message bank (3-5 summary lines; full bank saved to config), daily quota, and cadence (default: daily). One reply approves everything; corrections ride in the same reply.

Everything else is a SILENT DEFAULT — configured without asking, mentioned in at most one summary line each: default six-stage pipeline + inferred custom fields; auto segments; WideCast notification key reused from this client's existing config (WideCast Key Bootstrap); zeroed baseline; the campaign's automation task `{Client} - {Campaign} Daily Run` auto-created (paused/manual until the human runs or schedules it). The profile itself comes from the Solo Agency Profile Bootstrap with its single confirmation.

Progress display in Quick Start: a compact 3-step roadmap (list -> sendbox + identity -> goal confirmation) plus one line "everything else is configured automatically". The full 9-step roadmap applies only to standalone installs without Solo Agency setup.

## Latest Override: Setup Flow Does Not Send

The current OutreachCRM control-plane model supersedes any older "run the first report from setup" instruction:

- In Setup Flow, do not run the first agency run, first inbox sync, first enrichment pass, first draft batch, first send, tracking pull, or notification delivery.
- Setup Flow must finish by creating or resyncing the client-specific automation task and all persistent config that task reads to run correctly.
- The first send/enrich/draft/approval cycle must run in Automation Flow, using a client-specific task whose name begins with the client name, for example `AvenNgo - Buyer Leads Intro Daily Run`. The first run is that task's first execution.
- If the human asks to send, enrich, run a campaign, generate drafts, or "start emailing" while still in Setup Flow, verify/resync the automation task instead and tell the human the exact task name to run. Do not ask whether to send now, do not load `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` in the setup chat, and do not perform enrichment, drafting, sending, tracking, or notification.
- Any later setup/config change in this session must resync the Client Intelligence Profile, sendbox registry, list/campaign config, `schedule.md`, the automation manifest, the scheduled-run prompt, the native task body when editable, and `resync_log.md` (Automation Resync).

Setup completion means `ready_for_automation_first_run`, not `first_send_completed`.

## Source Preservation Rule

This file is detailed source material for the Stage 1 setup interview. Do not summarize away requirements, examples, checklists, schemas, protocols, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the human-facing reply, but it must still obey the full requirements in this file.

---

## The Setup Flow Send Refusal (exact wording)

When the human asks to send, enrich-and-send, run a campaign, or "email them now" while Setup Flow is active, the only valid response is to refuse the send, keep the session in Setup Flow, and hand off to the automation task. Use this wording (adapt the client name and task name only):

```text
I will not send email or run a campaign from this setup session. Setup Flow only
configures OutreachCRM — it never sends, never enriches a contact for send, and never
runs a campaign. Sending happens only in Automation Flow, and even there nothing leaves
until you approve the drafts in chat.

I will finish or resync `{Client Name} - {Campaign} Daily Run` instead (the very first
pass is just that same task's first execution). Once setup
reaches `ready_for_automation_first_run`, run that task. It will sync the inbox, enrich
the first batch, draft emails against the campaign goal, and post the drafts for your
review. I will send only the drafts you `approve`.
```

Then end the reply with an `**[ACTION REQUIRED]**` block that names the exact client-specific automation task to run, or, if the task is already ready, next-action guidance whose FIRST suggestion is running that exact task. Never end a send-request handoff with "let me know" or a bare report link.

Do not ask "Do you want me to send it now?". Do not load the Scheduled Run entrypoint in the setup chat. Do not enrich, draft, send, pull tracking, or notify in Setup Flow.

---

## Minimal Human Input Rule

At setup, the agent asks only for:

- Client name, if not already known.
- What the client sells and to whom: a product/service description, the client's expertise or business description, or a public website/profile URL.
- Target location only if location matters (local service area, jurisdiction, geo-limited offer) and cannot be inferred.

The agent must not ask the human to define, list, or fill in:

- `icp` (ideal customer profile — the people OutreachCRM will email on the client's behalf)
- `pain_points`
- `value_proposition`
- `offer`
- `proof_points`
- `brand_voice`
- `target_audience`
- `do_not_mention` / `negative_topics`
- `language` assumptions
- pipeline stages, custom fields, or the campaign goal shape

The agent infers all of these first, then shows them for correction.

Good first setup question:

Ask the exact canonical question from `OUTREACHCRM_PLAYBOOK.md` "First Human Question" verbatim (do not rephrase); then infer ICP/value-prop/voice and show for correction.

Good add-client question:

```text
Please give me the new client's name and what they sell — a product/service
description, expertise, or a public website/profile URL — plus the service area if
location matters. I will infer the rest and show you to correct.
```

Bad setup questions:

- "What industry are you in?"
- "Please list your ideal customer profile."
- "Please list all the pain points."
- "Please define your value proposition."
- "Please choose your pipeline stages and custom fields."
- "What should the campaign goal be?"

Exception: if the agent cannot infer a critical field after reasonable research (for example, whether the offer is B2B or B2C, or whether the service is geo-limited) and the field changes the direction materially, it may ask one concise follow-up question.

### Step-By-Step Setup Interview Rule

Setup is conducted step by step, not as one long questionnaire. The agent follows this loop:

1. Ask one minimal setup question (or advance to the next of the nine setup steps).
2. Wait for the human's answer.
3. Immediately infer everything that can be inferred from that answer.
4. Show the inference to the human, marked with its status and rationale.
5. Ask the next minimal question only after showing the inference.

The agent must not collect all setup answers first and only show reasoning at the end. Every follow-up question includes a short `What I inferred from your last answer` section before the next question.

The ICP the agent infers is the client's ideal customer — the recipients OutreachCRM will email — not the client themselves. Distilled ICP, pain points, value proposition, offer, proof points, and brand voice feed the email-writing skill (Stage 6) as the client-profile half of every draft (voice, offer, compliance).

Example inference block:

```md
What I inferred from your last answer:
- Business: residential mortgage brokerage
- Ideal customer (ICP): first-time home buyers, rate shoppers, and homeowners weighing
  a refinance, in and around Austin, TX
- Pain points: unclear affordability, fear of over-paying on rate, confusion about
  pre-approval, uncertainty about monthly payment
- Value proposition: a broker who shops many lenders for you, gives a real pre-approval
  fast, and explains the monthly-payment math plainly
- Offer: free rate quote + pre-approval consult
- Proof points (to verify): "we compare 20+ lenders", local Austin closings
- Brand voice: plain, reassuring, numbers-first, low-pressure
- Do-not-mention: none inferred yet
- Language: English (audience and reports)

Next question:
Did I read the ideal customer and the offer right? Correct anything wrong above, and
confirm the service area: Austin only, or the wider Central Texas region your clients
buy in? That changes how tightly I geo-target the first list.
```

---

## Inference-First Rule

The agent thinks, infers, and researches before asking. The agent must:

- Use existing files first (a prior profile, an earlier `pending` note) — including the sibling Solo Agency content-pipeline Client Intelligence Profile when it exists (see the Solo Agency Profile Bootstrap rule below).
- Use the client description or public website/profile URL.
- Use public web research (WebSearch/WebFetch/browser tool) when available.
- Use known market patterns for the offer type.
- Use target-location context.
- Draft assumptions instead of blocking; proceed with reasonable assumptions when optional fields are missing.

Each inferred setup field carries:

- `value`
- `status`
- `rationale`

Allowed status values:

- `provided_by_human`
- `inferred_by_agent`
- `discovered_from_source`
- `human_corrected`

Example:

```md
## icp
value: First-time home buyers, rate shoppers, and refinance-curious homeowners in Austin, TX.
status: inferred_by_agent
rationale: The client is a residential mortgage broker in Austin. These groups carry the
most urgent, time-sensitive questions about affordability, rate, pre-approval, and monthly
payment, which makes them the highest-intent cold-email audience.
```

Research honesty: do not invent proof points, awards, closings, or claims the client has not confirmed. Anything the agent could not verify is marked `to_verify` and surfaced in the correction step. A draft (later, in Automation Flow) may contain only claims the client confirmed or a contact dossier supports with an `evidence_url`; that discipline starts at the profile.

---

## Solo Agency Profile Bootstrap (read-only)

OutreachCRM shares its per-client workspace with the Solo Agency content pipeline. When the same client already completed the content-pipeline setup, that product's Client Intelligence Profile exists ONE level above OutreachCRM's subtree:

```text
daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/
  client_profile_{client_slug}_{business_slug}_{location_slug}.md      <- Solo Agency (content) profile
  outreach/
    client_profile_{client_slug}_{business_slug}_{location_slug}.md    <- OutreachCRM profile (this stage)
```

**Bootstrap check (mandatory):** BEFORE asking the Step-1 opening question for a new client, look for `daily-content-pipeline/clients/{client_slug}/*/client_profile_*.md` one level above the `outreach/` subtree (never inside it). Three outcomes:

- **No match** → run the interview exactly as written below. Nothing changes.
- **One match** → enter Bootstrap Mode.
- **Multiple matches** (several `{business_slug}_{location_slug}` workspaces) → ask the human one question — which workspace this outreach client corresponds to — then enter Bootstrap Mode.

**Bootstrap Mode** replaces the Step-1 QUESTION, never the Step-1 CONFIRMATION:

1. Read the content-pipeline profile. Use ONLY its business-context fields. IGNORE its `public_data_sources` / `private_data_sources`, keyword-bank, collector, and PDNA production/distribution sections entirely — those concepts are deleted from OutreachCRM (DESIGN §3 "Deleted components") and must not be imported, referenced, or re-created here.
2. Pre-fill the OutreachCRM profile from this mapping. Every pre-filled field keeps the `value` / `status` / `rationale` discipline: `status: discovered_from_source` with a rationale naming the source file path; the human's confirmation upgrades it (`human_corrected` on edits).

   | Content-pipeline field | OutreachCRM field | Transform |
   |---|---|---|
   | `business_description` | `business_description` | copy, condense |
   | `industry`, `sub_industry` | `industry`, `sub_industry` | copy |
   | `business_offer` | `offer` | copy; split into offer items |
   | positioning / offer promise | `value_prop` | rewrite in cold-email voice |
   | `target_audience` | `icp` | NARROW to who will receive email (titles, firm types, geography, disqualifiers); never copy verbatim |
   | `pain_points` | `pain_points` | re-rank by cold-email relevance |
   | `target_location` | `target_location` | copy |
   | `language` | `language.recipient_language` + `language.human_report_language` | map both explicitly |
   | content pillars, proposal/website research | candidate `proof_points` (marked `to_verify`), `brand_voice` seed | `inferred_by_agent` |

3. Do NOT ask the canonical First Human Question. Open with the FULL Step-1 inference block (all pre-filled + inferred fields, each with its status) and ask ONE `**[ACTION REQUIRED]**` confirmation per the Show Inference And Research Rule. The human confirms or corrects; they never re-type what the content-pipeline setup already established.
4. Record the bootstrap in the profile's `bootstrap` block (Stage 7 schema): source path, the source profile's `last_reviewed_date`, bootstrap date, and the pre-filled field list.
5. Bootstrap is a ONE-TIME SNAPSHOT, strictly one-way. Later edits to the content-pipeline profile do NOT flow here automatically, and OutreachCRM NEVER writes to the content-pipeline profile. If the human says the content side changed, re-run this rule as Setup Repair and show a diff for confirmation.
6. Interview Steps 3, 4, 5, 7, and 9 are NOT bootstrappable — sending identity (legally load-bearing, CAN-SPAM), sendbox credentials, the contact list, the WideCast key, and the schedule are still asked exactly as written.
7. Step 6 (first campaign) uses the bootstrapped profile to PROPOSE the full campaign goal and a seed message bank — see Step 6 and Stage 5 §1c. The human approves or edits; they do not author from scratch.
8. A source field whose own status shows it was never human-confirmed on the content side may still be used, but its rationale must say so (e.g. `from Solo Agency profile; status there: inferred_by_agent, not yet human-confirmed`) so the human's attention lands on it during confirmation.

---

## Show Inference And Research Rule

Anything inferred or researched must be shown to the human before it is saved as stable setup context. The agent shows:

- Inferred `icp`
- Inferred `pain_points`
- Inferred `value_proposition` and `offer`
- Inferred or discovered `target_location`
- Inferred `proof_points` (marked `to_verify` where unconfirmed)
- Inferred `brand_voice`
- Inferred `language` assumptions
- Any inferred `do_not_mention` / `negative_topics`
- Compliance notes (sending identity gaps, opt-out method)
- Assumptions and rationale

The agent then asks, in an `**[ACTION REQUIRED]**` block:

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**I need you to:** correct anything wrong in the profile above. If it looks right, I will
save it as the Client Intelligence Profile and use it for every future automation run.
**Reply with:** `looks right` or the specific corrections
**Why:** This profile becomes the client-side half of every email OutreachCRM drafts.
```

The agent must not ask the human to fill every field manually. It corrects on feedback, marks corrected fields `human_corrected`, and saves.

---

## The 9-Step Setup Flow Interview

Setup follows exactly nine steps, aligned to the visible setup roadmap. Do not introduce steps 10+. Steps that own another stage's config (sendbox, list, campaign, analytics) load that stage in full first (LOAD LEDGER) and write through its tooling — never by hand.

### Step 1 — Business, ICP, pain points, value proposition, and voice (one question, then infer)

Ask the single opening question — unless Bootstrap Mode is active (see the Solo Agency Profile Bootstrap rule), in which case skip the question and open directly with the pre-filled inference block. From the answer (or the bootstrapped profile) plus research, infer and show for correction:

- `business` (what the client sells) and `offer`
- `industry` and `sub_industry` (Stage 7 schema fields; inferred, never asked)
- `icp` (who OutreachCRM will email)
- `pain_points` of the ICP
- `value_proposition` and candidate `proof_points` (mark unconfirmed ones `to_verify`)
- `brand_voice` (tone, formality, do/don't)
- `language` (audience language, report language)
- `do_not_mention` / `negative_topics`
- `target_location` (ask only if it matters and cannot be inferred)

Show the inference block, take corrections, and hold the corrected values for the profile save at the end of Step 1. This is the only place the human describes the business in prose; every later step is a short confirm.

### Step 2 — Pipeline template + custom fields

Propose the default sales pipeline (six stages) and ask only whether to customize. The default `pipelines.json` (`default_sales`) carries per-stage `probability` and `sla_days`:

```json
{"pipelines":[{"id":"default_sales","stages":[
   {"id":"new_reply","probability":0.10,"sla_days":1},
   {"id":"engaged","probability":0.25,"sla_days":7},
   {"id":"meeting_booked","probability":0.50,"sla_days":7},
   {"id":"proposal_sent","probability":0.70,"sla_days":10},
   {"id":"won"},{"id":"lost"}]}]}
```

The full pipeline object (including the deterministic rules `r1`–`r6`) lives in Stage 13; at setup, write the pipeline via `tool crm-store`, never as a hand-edited file. `tool crm-store` exists (Phase 1) — write CRM records through it; hand-editing a `crm/` record file is a critical violation. A workspace carried over from an older Phase-0 install must run `<bridge> tool crm-store --client-dir <DIR> validate --rebuild-index` once (DESIGN §22 R3). Ask whether the client wants to rename or add stages, and whether they track any custom fields on contacts, accounts, or deals (for example `loan_type`, `brokerage`, `renewal_month`). Custom fields land in `contact.custom_fields` / account `custom_fields` / deal fields per the Stage 7 schema.

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**Approve one option:** `use default pipeline` / `rename/add stages` / `add custom fields`
**What I will do after approval:** save the pipeline and any custom fields to this
client's CRM config through `tool crm-store`.
**Why:** The pipeline decides how replies become deals and when nudge tasks fire.
```

Default is fine for almost every client — do not force customization.

### Step 3 — Sending identity (from-name, signature, physical address, unsubscribe method)

Capture the identity that appears on every email and the CAN-SPAM footer. This is the one step where the agent must ask, because the values are the client's real-world facts and are legally load-bearing:

- `from_name` — the display name recipients see (a real person is best; "The {Client} Team" is acceptable).
- `signature` — name, role, client business name, reply-to, optional phone/site.
- `physical_postal_address` — a valid physical mailing address (street or registered PO box). **Required for CAN-SPAM:** every commercial email must include a physical address. Setup cannot mark sending identity complete without it.
- `unsubscribe_method` — how recipients opt out. For the priority `@gmail.com` (App Password) path, sendboxes default to `plain_text_mode` (no open pixel, no link rewrite), so the baseline opt-out is a plain footer opt-out line plus a `List-Unsubscribe` mailto alias (`{box}+unsub-{token}@…`); a one-click HTTPS `/u/` unsubscribe requires the tracker worker on a domain the client controls (Workspace/custom-domain path, wired in Stage 8/Stage 11, later phase). Record which method applies.

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**I need you to:** confirm the sending identity — from-name, one-line signature, and a
valid physical mailing address for the legal footer. Tell me the reply-to address too.
**Reply with:** the from-name, signature, physical address, and reply-to
**Why:** US CAN-SPAM requires a physical address and a working opt-out in every
commercial email. I cannot mark the client ready to send without them.
```

Encode, do not merely note: the profile stores the address and opt-out method, **and this step ALSO
writes the machine-readable copy the send engine actually reads** — `config/sending_identity.json`
in the client dir:

```json
{"from_name": "…", "physical_mailing_address": "…", "unsubscribe_text": "(optional per-client wording)"}
```

`tool gmail` appends the CAN-SPAM footer (postal address + visible opt-out line) to EVERY
outgoing body from this file, and its presend gate **fails closed** (`missing_physical_address`)
when the file or the address is absent — a client without it cannot send at all. Stage 9 audits its
presence. Step-1 subjects must be truthful (no fake `Re:`/`Fwd:`) — recorded here as a guardrail,
enforced at draft and pre-send.

### Step 4 — Connect the first sendbox

Load Stage 2 (`playbooks/02_SENDBOX_SETUP.md`) in full (LOAD LEDGER) before writing any sendbox config. Two auth modes, one interface:

- **`app_password` — the priority path for `@gmail.com`.** SMTP send + IMAP read via Python stdlib; no OAuth, no 7-day token expiry, preserves our Message-ID. Requires 2-Step Verification on the Google account and an App Password. Consumer `@gmail.com` limits are documented and accepted: `plain_text_mode` (measure by reply, no pixel), roughly 20–50 cold emails/day/box, never the operator's primary Gmail, cold bulk risks suspension at volume — start low with tight personalization.
- **`oauth` — advanced (Google Workspace / custom domain).** Gmail API with scopes `gmail.send + gmail.readonly` only. The OAuth app must be **Internal** to avoid the 7-day refresh-token expiry; if forced External/testing, weekly re-auth becomes a scheduled day-6 `**[ACTION REQUIRED]**`, not an error.

Never ask for the Google password, cookies, or OTP. The App Password request is allowed because it is a scoped app credential, not the account password:

Include the full step-by-step below in this first ask — do NOT wait for the human to ask "how do I create an App Password?". Never ask them to paste the App Password into chat; it reaches the tool only through the `OUTREACHCRM_APP_PASSWORD` environment variable.

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**I need you to create a Google App Password (nothing pasted to me):**
  1. Open https://myaccount.google.com/security and sign in to the SENDING Gmail account.
  2. Under "How you sign in to Google", turn ON 2-Step Verification (App Passwords require it).
  3. Open https://myaccount.google.com/apppasswords (Google may ask you to sign in again).
  4. Type an app name, e.g. `OutreachCRM {Client}`, click Create — Google shows a 16-character code ONCE; copy it now.
  5. In your local terminal, set it (do NOT send it to me): export OUTREACHCRM_APP_PASSWORD="the 16-char code"
**If "App Passwords" is missing:** 2-Step Verification is not fully on, or the account uses a security key only, or an organization / Advanced Protection blocked it. Fix 2FA first — Google's guide: https://support.google.com/accounts/answer/185833
**Reply with:** the sending address only, e.g. `sendbox: {address}@gmail.com` (never the App Password itself)
**Why:** OutreachCRM sends and reads that mailbox over SMTP/IMAP with the App Password. It is scoped to this app, never your main Google password, and I never see it — the tool reads it from your environment variable.
```

`tool gmail auth` (Phase 1, present) does this for real: the human sets `OUTREACHCRM_APP_PASSWORD` in their shell, then runs `<bridge> tool gmail --client-dir <DIR> auth --sendbox <slug> --email <you@gmail.com>`, which verifies SMTP+IMAP, writes `sendboxes/{sendbox_slug}/credentials.json` (gitignored, chmod 600), registers the box in `sendboxes/sendboxes.json` (`auth_mode`, `email`, `domain`, `quota_today`, `warmup_stage: week_1`), baselines the IMAP cursor, and flips the box to `status: healthy`. Do not improvise a one-off connectivity script, and never ask for the App Password in chat or as a CLI argument (env var only). Setup never sends a test cold email.

### Step 5 — Import the first list

Load Stage 3 (`playbooks/03_IMPORT_LIST.md`) in full (LOAD LEDGER) before importing. Accept a CSV/TXT/XLSX list. The importer mints a ULID `lead_id` per row (email is NOT required — a contact may be name + a profile URL only), maps and normalizes fields, dedupes across identities, and checks every identity against global and client suppression at import time. Output lands under `lists/{list_slug}/` (`list_manifest.json`, `leads.jsonl`, `import_log.md`) and creates contacts through `tool crm-store`.

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**I need you to:** give me the first list — a CSV/TXT/XLSX path or paste the rows. Any
columns you have (name, email, company, phone, profile URL, city) are fine; email is not
required.
**Reply with:** the file path or the pasted list
**Why:** This becomes the client's first contact set. I import and dedupe it now; I do
not enrich or email anyone in Setup Flow.
```

Setup imports and dedupes only. It does not verify-enrich for send (Stage 4) and does not draft or send. Enrichment and drafting begin in the first automation run.

### Step 6 — Create the first campaign with a structured goal

Load Stage 5 (`playbooks/05_CAMPAIGN_MANAGEMENT.md`) in full (LOAD LEDGER) before writing `campaign_config.json`. The goal is the writing blueprint, not a label — it drives what the agent writes. Infer a proposed goal from the profile and show it for correction rather than asking the human to author JSON:

```json
{"campaign_slug":"",
 "goal":{"goal_type":"book_meeting|get_reply|direct_sale|reactivation|nurture_upsell|event_invite",
   "objective":"","offer":"","value_proposition":"",
   "proof_points":[{"claim":"","evidence_url":""}],
   "cta":{"type":"reply_yes|link|calendar","text":""},
   "companion_doc":{"instructions":"","on_fail":"skip|default_link","default_link":""},
   "message_bank":[{"msg":"","source":"operator|agent","approved":true}],
   "success_event":{"on":"reply_positive","create_deal_stage":"new_reply"}},
 "audience":{"segment":"","personalization":{"required_hook_types":[],"min_confidence":0.7,
   "no_hook_fallback":"skip|generic_honest_opener"}},
 "sequence":[{"step":1,"intent":"hook + offer, one CTA","tracking":"plain_text"},
   {"step":2,"gap_days":4,"intent":"deliver new value"},
   {"step":3,"gap_days":5,"intent":"social proof"},
   {"step":4,"gap_days":7,"intent":"breakup"}],
 "sendboxes":[],"daily_quota":40,"approval_mode":"manual_all",
 "guardrails":{"banned_claims":["guarantees"],"no_fake_re":true},
 "channel_strategy":"email_first"}
```

`goal_type` shapes the email: `book_meeting` → short, one time-bound CTA; `get_reply` → ends with a question, no link; `direct_sale` → value + one offer link; `reactivation` → evidence of a prior relationship. Every final step is a breakup. `success_event` wires straight into the CRM rules engine (a positive reply creates a deal at `new_reply`). Default `approval_mode: manual_all` even for bumps.

The Stage-5 intake (`05_CAMPAIGN_MANAGEMENT.md` §1b/§1c) adds TWO questions you must ask here too:
(1) *"Is there a companion document/link? Describe how to get the link"* — and if declared, its
failure policy (`on_fail: skip|default_link`) MUST be asked when the operator forgets; (2) *"List
every key message for email 1 and the follow-ups (or ask me to propose a set)"* — then EXPAND the
operator's list with domain knowledge and show the full bank back for approval before creating the
campaign. Store both on the goal as shown above.

In Bootstrap Mode (Solo Agency Profile Bootstrap), propose BOTH before asking: (a) the complete
goal JSON — `goal_type` (a value from the valid enum), `objective`, `offer`, `value_proposition`,
`proof_points`, `cta` — derived from the bootstrapped profile; and (b) a seed `message_bank` drawn
from the profile's value proposition, pain points, and the content-pipeline positioning/themes,
every proposed entry tagged `source: "agent"`. The two intake questions then collapse into
confirmations: the companion-doc question is still asked normally (operator knowledge, never
inferable), and the message-bank question becomes *"here is the proposed bank — approve, trim, or
add your own"*. The Stage-5 §1c hard gate is unchanged: the operator approves every agent-added
entry before the campaign is created.

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**Approve one option:** `use this goal` / `change goal_type` / `edit offer or CTA`
**What I will do after approval:** save the first campaign config. I will NOT enrich,
draft, or send in this session.
**Why:** The campaign goal decides what every email in this sequence tries to achieve.
```

Setup writes the campaign config only. It does not build the enrich queue, draft, preview, or send.

### Step 7 — PDNA notification (WideCast API key only)

In OutreachCRM, PDNA is **notification-only**. There is no production, rendering, or distribution stage — WideCast is used solely to deliver operator notifications: a Telegram message (`sendNotification`) with the report link, an email fallback when Telegram is unavailable, and an optional `uploadAsset` call to host the report file behind the link. This is the one provider setup in Setup Flow, and it is client-scoped.

**Read existing config before asking (WideCast key bootstrap).** Do NOT ask for the WideCast key if a client-scoped key already exists for THIS client:

1. Check OutreachCRM's own provider config (`.../outreach/integrations/providers/provider_config.local.json`). If it is already configured and the account verifies, notification is connected - do not ask again.
2. Otherwise check the sibling Solo Agency content-pipeline provider config for the SAME client (`integrations/providers/provider_config.local.json`, one level ABOVE the `outreach/` subtree). If a client-scoped WideCast key (`api_key_env`/`api_key_local`) exists and verifies there, REUSE it: write OutreachCRM's own `outreach/integrations/providers/provider_config.local.json` referencing the same key (the operator's own WideCast account, already connected for this client), verify the account + notification capability, send the confirmation ping, and mark notification connected - WITHOUT asking the human to paste the key again. Tell the human you reused the WideCast key they already connected for this client.
3. Only if neither config has a client-scoped key for this client do you ask the human (the `**[ACTION REQUIRED]**` below).

This one-way read of the SAME client's sibling config is allowed - it is client-scoped, the operator's own verified key (mirrors the Stage-1 profile bootstrap). It is NOT the forbidden case of adopting a global MCP/native account as this client's connection; that stays forbidden, and a global account is never auto-reused (`global_mcp_not_client_scoped`).

If no client-scoped key exists for this client yet, ask only for the WideCast API key; the agent configures, verifies, discovers, and resyncs the rest. Provider config lives per client under `integrations/providers/provider_config.local.json`. The key is referenced by `api_key_env` (an environment-variable name) or `api_key_local` (a path/handle) — **never a field literally named `api_key`**. Capability discovery uses the OpenAPI helper `tool provider`, caching to `provider_openapi_cache.yaml` and recording capabilities in `provider_capabilities.json`.

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**I need you to:** paste this client's WideCast API key so operator notifications
(email + Telegram) work.
**Reply with:** `widecast key: {key}`
**Why:** After a run, OutreachCRM sends you the run summary and report link via WideCast
`sendNotification`. It is notification-only — no video, rendering, or distribution.
```

Do not ask provider/scope/spend/publish/account-identity questions — those belong to a non-default stack the human must explicitly request. Do not treat a global MCP/native provider account as this client's notification connection. If the current session cannot write provider config, hand the API-key action to the setup/automation task via an `**[ACTION REQUIRED]**` block. WideCast is operator-only; it never appears in client-facing output (the scrub gate strips it).

### Step 8 — Baseline analytics (nothing sent)

Load Stage 12 (`playbooks/12_TRACKING_ANALYTICS.md`) reference and initialize the client's analytics baseline with everything zeroed, because nothing has been sent. Create `analytics/metrics_log.md` and an empty `analytics/learning_log.md`. Record the honest metric model up front: reply / bounce / unsubscribe are exact; open is an estimate (labeled "estimated"); click is fairly reliable after bot filtering. For `@gmail.com` `plain_text_mode` boxes there is no open pixel at all, so the baseline is reply-measured.

Setup only records that no sends exist yet and that the learning loop starts empty; it runs no tracking pull and computes no rates.

### Step 9 — Create the automation task(s)

This is where Setup Flow ends. Create or resync:

- **One client-specific automation task**, name beginning with the client name, for example `AvenNgo - Buyer Leads Intro Daily Run` (the first pass is this task's first execution, not a separate task). Its prompt pins `target_client_slug` so it can never touch another client. Configure the schedule/cadence the human wants (daily, weekday, manual-only, first-run-only).
- **One agency-wide maintenance task** `OutreachCRM - GitHub Update Watch`, offered after the client task exists. It is barred from client-facing channels and does not touch client data (Stage 11).

Write every persistent state file the next automation run reads: the Client Intelligence Profile, `sendboxes/sendboxes.json`, list/campaign config, `daily-content-pipeline/schedule.md`, the automation manifest, the scheduled-run prompt, the native task body when editable, and `resync_log.md`. Then run the dry-read Automation Resync verification.

End with an `**[ACTION REQUIRED]**` block naming the exact task to run for the first pass and stating that setup is `ready_for_automation_first_run`. Do not send, do not run the task from the setup chat, and do not ask "run it now?".

---

## Client Intelligence Profile

The Client Intelligence Profile is the durable, client-scope record that every automation run reads and every draft is built from. Its canonical field schema lives in Stage 7 (`playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`); this stage owns the interview that fills it. Path (per the on-disk layout, `daily-content-pipeline/`):

```text
daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outreach/
  client_profile_{client_slug}_{business_slug}_{location_slug}.md
```

Stage-1-relevant fields the interview must fill (each with `value` / `status` / `rationale` where inferred):

- `client_name`, `client_slug`, `business_slug`, `location_slug`
- `business`, `industry`, `sub_industry`, `offer`, `value_proposition`, `proof_points[]` (each `to_verify` until confirmed)
- `icp`, `pain_points[]`, `target_location`
- `bootstrap` (only when Bootstrap Mode was used): source profile path, source `last_reviewed_date`, bootstrap date, pre-filled field list (Stage 7 schema)
- `brand_voice`, `language` (audience language, report language)
- `do_not_mention[]` / `negative_topics[]`
- `sending_identity`: `from_name`, `signature`, `physical_postal_address`, `reply_to`, `unsubscribe_method`
- `pipeline_id` (default `default_sales`) and any `custom_fields`
- `sendbox_refs[]` (slugs into `sendboxes/sendboxes.json`)
- `default_campaign`: `campaign_slug` + goal summary
- `notification`: WideCast provider reference (`api_key_env`/`api_key_local`, capabilities), notification-only
- `analytics_baseline`: nothing sent; metric-honesty note
- `setup_state`: `ready_for_automation_first_run`
- `automation_task_name`: `{Client Name} - {Campaign} Daily Run` (the first pass is this task's first execution)

Slug rules: lowercase, hyphens, no punctuation or spaces. The client folder key is `{client_slug}/{business_slug}_{location_slug}`. All monthly artifacts (activities, sent log, campaign history, reports, inbox sync, outputs) use `YYYY-MM/` folders. The profile is written through `tool crm-store`/the Stage 7 tooling where the schema requires it; do not hand-edit CRM record files, which is a critical violation. `tool crm-store` exists (Phase 1) — write CRM records through it; hand-editing a `crm/` record file is a critical violation. A workspace carried over from an older Phase-0 install must run `<bridge> tool crm-store --client-dir <DIR> validate --rebuild-index` once (DESIGN §22 R3). (The Client Intelligence Profile itself is a `.md` file, never a crm_store collection, so it was always a direct write.)

---

## Multi-Client Batch Mode

OutreachCRM supports many clients across many offers. If the human already has several clients, accept a compact list.

Example human input:

```md
I manage 5 clients. Set up one workspace for each:

1. Smith Law - DUI defense - Los Angeles - list: dui-leads.csv - goal: book_meeting
2. Austin Home Group - real estate - Austin, TX - list: buyers.csv - goal: get_reply
3. Bright Mortgage - home loans - Texas - list: refi-list.xlsx - goal: direct_sale
4. Miami Shield Insurance - home + auto - Miami - list: renewals.csv - goal: reactivation
5. Vienna AI Ops - automation agency - Vienna - list: saas-founders.csv - goal: book_meeting
```

The agent must:

1. Create one client workspace folder per client under `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outreach/`.
2. Infer the full profile (ICP, pain points, value proposition, offer, voice) for each client and show a per-client summary.
3. Ask the human to correct only what is wrong (one `**[ACTION REQUIRED]**` block, batched).
4. Save one Client Intelligence Profile per client.
5. Add every client to `clients_index.md`.
6. Configure each client's sendbox, list, first campaign goal, sending identity, and notification per the nine steps — each client gets its own sendbox(es), suppression, and data (no shared quota, no cross-client visibility).
7. Configure the schedule and create one automation task per client, each pinning its own `target_client_slug`.
8. Reach `ready_for_automation_first_run` for each client. Send nothing.

If entries are incomplete, infer what is possible and ask only for the missing critical items (a sendbox to send from, a physical address for the footer, a list). Never fabricate a workspace: if the client name or enough business context is missing, ask for it and keep the root ready.

---

## Incremental Client Onboarding Rule

OutreachCRM supports starting with zero clients and adding clients over time. The human is not required to provide all clients at once.

If there are no clients yet, create only the data root scaffold and enter First Client Setup Mode:

```text
daily-content-pipeline/
  clients_index.md
  schedule.md
  storage_config.json          # {"backend":"json"}
  provider_defaults.json       # WideCast notification catalog, no secrets
  suppression/global_suppression.jsonl
  clients/
  automation/
  notifications/
```

First Client Setup Mode is the same as Add Client Mode, triggered automatically during the first run when `clients_index.md` has no real client rows. Proceed as far as possible toward setting up the first client instead of stopping after root creation.

In First Client Setup Mode, ask only:

- Client name, if not already known.
- What the client sells and to whom (product/service, expertise, or public website/profile URL).
- Target location only if it matters and cannot be inferred.

Do not create fake client workspaces. A public website/profile URL counts as business context if it is publicly accessible and gives enough to infer the profile.

Whenever the human says something like "Add a new client", "Add this client", "We just got a new client", or "Add client: …", enter Add Client Mode. In Add Client Mode ask only for missing critical information (name; what they sell; location if it matters), then infer `icp`, `pain_points`, `value_proposition`, `offer`, `proof_points`, `brand_voice`, `language`, `do_not_mention`, and a proposed pipeline + campaign goal.

Then follow the same nine-step setup model. Do not introduce Add Client setup steps 10+.

1. Show the inferred profile summary and ask the human to correct only what is wrong.
2. Create/update the client workspace folder, Client Intelligence Profile, CRM folders (`accounts/`, `contacts/`, `deals/`, `activities/YYYY-MM/`, `tasks/`, `pipelines.json`, `suppression.jsonl`), `lists/`, `campaigns/`, `analytics/`, `reports/`, `outputs/`, and the `clients_index.md` row.
3. Save the inferred ICP, pain points, value proposition, offer, brand voice, language, and compliance notes.
4. Save the pipeline (default six-stage) and any custom fields, plus the first campaign goal.
5. Connect the first sendbox (Stage 2), import the first list (Stage 3), and capture the sending identity (from-name, signature, physical address, unsubscribe method).
6. Configure the recurring schedule and create/resync the client-specific automation task whose name begins with the client name, for example `Nguyen Law - Consult Intro Daily Run`.
7. Set up PDNA notification (WideCast API key only, notification-only) when the human provides the key; otherwise mark it pending and hand the API-key action to the automation task.
8. Initialize the analytics baseline (nothing sent).
9. Reach `ready_for_automation_first_run`. Do not enrich, draft, send, pull tracking, or notify inside Setup Flow.

Example:

Human:

```md
Add this client: Nguyen Law, immigration lawyer in San Jose. Goal: book consults.
Sendbox: intake@gmail.com (App Password). List: sj-immigration-leads.csv.
```

Agent creates:

```text
daily-content-pipeline/
  clients/
    nguyen-law/
      immigration-law_san-jose/
        outreach/
          client_profile_nguyen-law_immigration-law_san-jose.md
          sendboxes/sendboxes.json
          lists/{list_slug}/list_manifest.json  leads.jsonl  import_log.md
          crm/{accounts,contacts,deals,activities/YYYY-MM,tasks,pipelines.json,suppression.jsonl}
          campaigns/{campaign_slug}/campaign_config.json
          analytics/metrics_log.md  learning_log.md
          approvals/approval_log.md
          reports/  outputs/YYYY-MM/  outputs/latest/
          integrations/providers/provider_config.local.json
```

The agent configures the schedule, prepares the `Nguyen Law - Consult Intro Daily Run` task pinning `target_client_slug: nguyen-law`, runs Automation Resync, and tells the human the exact task name to run for the first pass. Setup Flow does not enrich, draft, or send Nguyen Law's first batch.

---

## Mandatory Automation Readiness Protocol

This protocol applies after the first client setup, after adding a new client, and after repairing an incomplete Client Intelligence Profile. Setup Flow is not a menu of optional next steps. The agent must not ask the human to choose between connecting a sendbox, importing a list, configuring the schedule, running the first send, or sending "just one test".

The correct order is fixed:

1. Finish the profile interview and save the Client Intelligence Profile.
2. Save the pipeline (default six-stage) and any custom fields.
3. Capture the sending identity (from-name, signature, physical address, unsubscribe method).
4. Connect and verify the first sendbox (Stage 2). No cold send.
5. Import and dedupe the first list (Stage 3). No enrich-for-send.
6. Save the first campaign goal (Stage 5).
7. Configure PDNA notification (WideCast API key only, notification-only), or mark it pending.
8. Initialize the analytics baseline (nothing sent).
9. Configure the schedule and create/resync the client-specific automation task (name begins with the client name; prompt pins `target_client_slug`), then offer the agency-wide `OutreachCRM - GitHub Update Watch` maintenance task.

Every persistent state file the next automation run reads must be current before completion: Client Intelligence Profile, `sendboxes/sendboxes.json`, list/campaign config, `provider_config.local.json`, `daily-content-pipeline/schedule.md`, the automation manifest, the scheduled-run prompt/task body, and `resync_log.md`. Run the dry-read Automation Resync verification before calling setup complete.

End Setup Flow with `ready_for_automation_first_run`, not `first_send_completed`. Enrichment, drafting, preview, sending, tracking, and notification run only in Automation Flow.

### First Automation Run Contract

The first send/enrich/draft/approval cycle happens in the client-specific automation task, after profile save, sendbox connect, list import, campaign-goal save, schedule setup, task creation, and a human action to run or schedule that task.

- Setup Flow must not build an enrich queue, draft, preview, or send merely to "get started".
- The first automation run follows the Daily Run order (orchestrated by `playbooks/SCHEDULED_RUN_ENTRYPOINT.md`, not by this stage): sync inbox → pull tracking → triage + apply CRM rules → follow-up advising → load new pipeline (verify → enrich → step-1 draft into `pending_approval`) → send only `outbox/approved/` within quota → compile Today View → reports → notify → Stage 9 audit.
- Nothing is sent until the operator reviews the Approval Report and replies `approve …` in chat. `approval_mode` defaults to `manual_all`, even for bumps. Rejections feed `learning_log` for the next batch.
- Suppression is checked at every send-capable path (initial, follow-up) and at import against all identities. If the tracker pull has not succeeded within the configured window, sending for that box is blocked so opt-outs cannot sit unhonored.
- The first automation run's operator notification (WideCast `sendNotification`, email fallback) reports run status, drafts awaiting review, the report link, blockers, and any required action, and is logged in `notifications/notification_log.md`. The client-facing weekly report (the only scrubbed, client-facing output) is not produced in Setup Flow.

The automation-run handoff message must show a compact progress roadmap and end with a concrete next-step `**[ACTION REQUIRED]**` (approve the drafts, re-auth a box, or paste a missing API key). Do not end with only a report link, a summary, or "let me know".

Good first-automation-run chat pattern:

```md
The first automation run for {Client Name} is done.

Drafts awaiting your review: {N}  (High confidence {H} / Review carefully {R})
Approval Report (operator-only): {absolute path or link}

OutreachCRM setup + automation progress
✓ 1. Profile: business, ICP, pain points, value proposition, voice
✓ 2. Pipeline (default six-stage) + custom fields
✓ 3. Sending identity (from-name, signature, physical address, opt-out)
✓ 4. First sendbox connected ({address}@gmail.com, App Password)
✓ 5. First list imported and deduped ({M} contacts)
✓ 6. First campaign goal saved ({goal_type})
✓ 7. PDNA notification (WideCast) connected — notification-only
✓ 8. Analytics baseline set (nothing sent before this run)
✓ 9. Automation task created: `{Client Name} - {Campaign} Daily Run`
→ First run drafted {N} emails; none sent yet — waiting for your approval.

Automation freshness check: ✓ latest approved changes synced into the automation/
scheduled task prompt, config, and state; tomorrow's scheduled run will load the newest state.
```

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**I need you to:** review the drafts and approve what should send.
**Reply with:** `approve all`, `approve 1-20, 35`, `reject 7: reason`, `edit 12: ...`, or `hold 5`
**Why:** Nothing leaves OutreachCRM without your explicit approval.
```

Bad patterns (all forbidden): ending with only a report link; asking "Do you want me to send it now?" in Setup Flow; sending before an `approve`; hiding the required question inside a Markdown file or a long paragraph.

### Automation Resync Rule

Any post-setup configuration change (new sendbox, changed schedule, edited campaign goal, new physical address, added custom field, provider key change) must re-sync the profile, `schedule.md`, the automation manifest, the scheduled-run prompt, the native task body when editable, and `resync_log.md`, then verify with a dry read that the next scheduled run will load the newest state — before the change is called complete. A change that updates one file but not the task the scheduler runs is not complete.

### Schedule Rule

Ask cadence questions after the profile, sendbox, list, and campaign goal are known and before the client-specific automation task is marked ready. Ask whether the human wants daily, weekday-only, multiple-times-daily, manual-only, first-run-only, or another cadence, then write `daily-content-pipeline/schedule.md`, the automation manifest, the scheduled-run prompt/task body, and the relevant config. During Setup Flow, do not offer to run the first pass immediately; finish by preparing or resyncing the client-specific automation task whose name begins with the client name. A `run now` / `manual run` / `test run` request is honored only in Automation Flow (it bypasses the schedule window for the target client only) — never as a way to send from the setup chat.

---

## Setup Repair Mode

If a Client Intelligence Profile exists but is incomplete, stale, or inconsistent:

1. Infer missing values where possible.
2. Research missing values where possible.
3. Show proposed repairs to the human in an `**[ACTION REQUIRED]**` block.
4. Ask the human to correct only what is wrong.
5. Update the Client Intelligence Profile (through the Stage 7 tooling where the schema requires it).
6. Run Automation Resync so the client-specific automation task reads the repaired state.
7. Continue toward `ready_for_automation_first_run` — do not send from repair mode.

Do not discard existing human-provided values unless the human confirms. If repair suspects stale playbooks or code, load Stage 11 (`playbooks/11_UPDATE_AND_VERSION_WATCH.md`) with a LOAD LEDGER and follow the fresh-source protocol before trusting local files.

---
