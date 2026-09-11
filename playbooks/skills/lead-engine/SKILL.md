---
name: solo-agency-lead-engine
description: >-
  Use whenever the human wants to FIND, HUNT, or COLLECT leads/prospects from
  data sources — e.g. "find people who need insurance", "find realtors in
  Orange County", "who is asking for a mortgage in my groups", persona-based
  prospecting, or any open-ended "go get me N leads" request. This is the
  capability-driven ACQUISITION orchestrator: it reads the Local Collector's
  live capability catalog, plans a sequence of collection jobs for a fuzzy
  intent, runs an autonomous gather loop until a lead KPI or a safety stop, then
  hands the collected items to Stage 10 for scoring, reporting, and storage. It
  does NOT replace Stage 10 (classification/report/storage) — it feeds it.
---

# Lead Engine — Capability-Driven Lead Acquisition

Sub-stage: `10A` (loads after `playbooks/10_LEAD_COMPETITOR_DETECTION.md`)

## Load Rule

Load whenever the human asks the agent to go and FIND/COLLECT leads for an
open-ended intent that is not already a single named source scan — anything of
the shape "find me people who {need/want/are looking for} X", "get me ~N leads",
"prospect this persona", or "search my groups for buyers". Always load Stage 10
alongside this skill; this skill acquires candidate items, Stage 10 qualifies
and reports them.

## What this skill is (and is not)

- **Is:** the planner + autonomous loop that turns a vague human intent into a
  concrete sequence of Local Collector capability jobs, runs them, deepens via
  pagination, and stops at a KPI or a safety limit. It also runs the row-level
  classification pass (see "Classification (extractor tier)" below) using
  Stage 10's own rule file — not a restatement of it.
- **Is not:** the report or the storage schema. Lead definitions, scoring, the
  value-first comment rules, the HTML report contract, and the storage schema
  all live in Stage 10 and are authoritative. Do not restate or override them.

## Hard Gates

- **Read the catalog first, every run.** Never assume the toolset. `GET
  http://127.0.0.1:17321/capabilities` and plan only over capabilities whose
  `status` is `stable` or `beta`. Treat each entry's `when_to_use`, `inputs`,
  and `output_schema` as the contract (this is the MCP-style discovery the whole
  design depends on).
- **Read-only collection only.** The collector is read-only. This skill NEVER
  joins groups, sends messages, comments, follows, or performs any write/outreach
  action. Joining a group is a human action — see `safety.md`.
- **Obey Stage 10's outreach ban.** No auto-DM, auto-comment, contact scraping,
  or outreach. Leads are review signals for the human, not permission to contact.
- **Obey the safety envelope.** Every run is bounded by a KPI and the ban-risk
  stop conditions in `safety.md`. When in doubt, stop early and report what was
  gathered plus why it stopped.
- **Hand off, don't fork.** Collected candidate items are passed to Stage 10's
  detection workflow (§Detection Workflow, required fields, scoring) and stored
  via Stage 10's ledger (`history/YYYY-MM/lead_competitor_opportunities.jsonl`).

## Classification (extractor tier)

Every candidate item (a post, comment, caption, or people-search row from step 3 of the loop)
is qualified by applying `playbooks/LEAD_QUALIFICATION_RULE.md` **verbatim**, Step 1 first, as the
classifier's instruction — this is Stage 10's rule; the lead-engine skill loads and runs it, never
restates it. The classifier also reads the client's `buyer_profile` block (`sells`, `sells_to`,
`types`, `why_they_need`, `location`, `competitors`, `not_buyers` — the Client Intelligence Profile,
Stage 10 §Definitions) and outputs exactly the rule's own JSON per row: `{id, person_type,
sells_to_match, fit, fit_reason, intent, intent_reason, decision}`. Step 1 first means WHO this
person is is decided before intent — a person of the right type with no stated need is `warm`,
never dropped for lacking a need phrase.

The rule was validated blind against 120 scenarios × 5 clients with Haiku (96.2% lead/non-lead,
93.8% exact tier), so the **extractor tier** — the runtime's smallest capable model
(`playbooks/TEAM_MODEL.md`) — is the right tier to run this pass; do not escalate to a bigger model
by default. A hot-vs-warm borderline call MAY be escalated one tier for a second read, but the FIT
decision (rule Step 1 — who this person is) is never escalated or overridden by a bigger model;
if fit reads medium/low, that answer stands.

## The loop (general solver)

This is the method for ANY open-ended lead request. It is deliberately generic
because user requests are unbounded.

```text
0. DISCOVER   GET /capabilities  → know the available tools (ids, when_to_use, inputs, output_schema, status)
1. INTERPRET  turn the human intent into: { target persona (a line of the client's buyer_profile.types),
              per-channel terms (recipes.md's channel table — role/product primary, intent only when
              anchored), industries, location, KPI N }
              (ask the human only for what you cannot infer: the KPI and any missing offer/persona)
2. PLAN       pick a recipe from recipes.md, or compose a capability sequence yourself using the catalog.
              A plan is an ordered list of { capability, inputs, why }.
3. GATHER LOOP (bounded by safety.md):
     for each planned step:
        submit a collector job (POST /jobs/run_now) with the capability + inputs
        read the run's records (private_data_points.jsonl → .records.items)
        classify each item ("Classification (extractor tier)" above) → keep only decision hot|warm|watch;
        dedupe by profile/post URL
        accumulate qualified leads
     re-evaluate after each step:
        - KPI reached?                → STOP (success)
        - diminishing returns (a step adds ~0 new qualified leads twice)? → widen (new keyword/source) or STOP
        - safety trip (see safety.md)? → STOP (report the reason)
        - more depth available and under budget? → DEEPEN: re-run the productive capability with a higher
          inputs.max_pages (cursor replay) or advance to the next planned source
4. QUALIFY    Stage 10 scores each lead (hot|warm|watch, confidence) and drafts a value-first comment.
5. REPORT     Stage 10 HTML report contract + storage ledger. Always disclose: how many leads, from which
              sources/keywords, how many pages/scrolls, and why the run stopped (KPI vs safety vs dry).
```

## Capability-driven planning (how to map a fuzzy intent → tools)

The agent behaves like an MCP client: it does not hard-code which capability to
call; it matches the request against the catalog.

1. Extract from the request: the **persona** (which `buyer_profile.types` line — who they are, not
   an activity the offer enables), the **per-channel terms** (role/product/stage terms primary,
   occupation+place for people search, jargon+hashtags for IG/X search, intent terms only when
   anchored to the role or the offer — `recipes.md`'s channel table, sourced from `00_CORE_CONTEXT_
   REQUIREMENTS.md`'s keyword-bank section), the **container** (where they gather: groups, search,
   a profile's network), the **location**, and the **KPI** (how many leads; ask if unstated).
2. Map to capabilities by reading each `when_to_use`:
   - "find where they gather" → `fb.groups.search` (communities), `fb.people.search` (persona-by-occupation+place).
   - "read what they post / find intent" → `fb.group.search_posts` (role/product/stage/intent terms
     inside a group, per-kind quota), `fb.group.posts` (a group's recent feed), `fb.profile.posts`.
   - "map a known person's network" → `fb.profile.friends` → then `fb.people.search` per friend for industry (see the catalog note on inferring industry).
   - deep coverage → set `inputs.max_pages` (cursor replay) instead of hoping scroll loads more.
3. If no stable/beta capability fits the intent, say so plainly and either fall
   back to a generic source scan (Stage 2/8) or tell the human what capability
   is missing — do not silently pretend a screen is covered.

## Output contract

The run's deliverable is Stage 10's report + ledger, plus a short run summary:

```text
Gathered {K} qualified leads for "{intent}".
Sources: {groups/searches used}. Keywords: {intent keywords}.
Bank updated: {terms written back to the per-source keyword bank, with hits/leads} — the daily monitoring searches these from tomorrow (`recipes.md` step 6).
Depth: {pages/scrolls}. Stopped because: {KPI reached | safety limit: <which> | dry: no new leads}.
Next: {suggested widen/deepen options for the human}.
```

## Files in this skill

- `recipes.md` — ready-made capability sequences for the common intents. Recipe A is the canonical
  Facebook leg of the Social Discovery Pass (`playbooks/10_LEAD_COMPETITOR_DETECTION.md` step 11C):
  fixed order feed → people → groups → in-group. The pass also runs the same shape on Instagram
  (search → people → profile depth → comments) and X (search Latest → people → profile depth →
  replies), interleaved round-robin across all three platforms — see Stage 10's "Social Discovery
  Pass" for the full platform table and the round-robin rule. Its discovery terms come from
  `tool public-keywords ... plan --kind discovery` (a dedicated `community_discovery` kind, shared
  across all three platforms, separate from the intent bank `recipes.md` steps 3/6 use), and
  Facebook's ranked candidate groups persist to `history/YYYY-MM/facebook_discovery_shortlist.jsonl`
  (fields and job shapes in `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`) — Instagram and X
  have no groups, so nothing is added there for them.
- `safety.md` — the KPI + ban-risk stop conditions, the join-is-human rule, and
  the ToS/privacy boundaries this loop must never cross.
- `playbooks/LEAD_QUALIFICATION_RULE.md` — the Fit × Intent rule this skill's classifier applies
  verbatim (see "Classification (extractor tier)" above); re-run `playbooks/tests/lead-rule` after
  any edit to it, per that file's own mandatory-test note.
- Scoring, lead schema, comment rules, report + storage: **Stage 10**
  (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`) — authoritative, do not duplicate.
