# Running the lead-rule regression test

Mandatory after any edit to `playbooks/LEAD_QUALIFICATION_RULE.md`. Do not
ship a rule edit without a fresh run of this procedure showing it does not
score below `baseline/baseline_metrics.json`. All paths below are relative
to this directory (`playbooks/tests/lead-rule/`).

## 0. Before you start

You need `playbooks/LEAD_QUALIFICATION_RULE.md` (the rule text you're
scoring) and `clients.json` (in this directory — 5 fictional buyer
profiles, format only, never edited for a test run). Pick a date string for
this run, e.g. `2026-09-15`; everything below writes under `runs/<date>/`.

## 1. Split the dataset into blind chunks

```
python3 split_blind.py --dataset dataset.json --out-dir runs/<date>/blind --chunk-size 10
```

This writes `runs/<date>/blind/chunk_01.json` .. `chunk_12.json` (10
scenarios each) and `runs/<date>/blind/map.json` (blind id `s001` -> real
id `A01`). Each blind scenario carries only `id, platform, surface,
community, author_line, text, date_note` — no group, no trap, no expected
label. The judge never sees those.

## 2. Spawn one Haiku sub-agent per (client x chunk)

5 clients x 12 chunks = 60 sub-agent calls. Each call reads exactly one
chunk file and one client's profile, and writes one judgments file. Spawn
them with `model: "haiku"` — the rule is only proven if the model that
actually runs it in production is the one judging it. Do not use a bigger
model "to get cleaner numbers" — that defeats the test.

Exact prompt template (fill in `<CLIENT_KEY>` and the two file paths):

```
You are blind-judging a batch of items against a fixed rule. Read these
two files in full before answering — do not summarize them, do not add
your own judgment of what the "right" answer should be beyond what the
rule says:

1. The rule: playbooks/LEAD_QUALIFICATION_RULE.md
2. The client's buyer profile: the entry with "key": "<CLIENT_KEY>" in
   playbooks/tests/lead-rule/clients.json

Then read every item in this batch file:
playbooks/tests/lead-rule/runs/<date>/blind/chunk_<NN>.json

For each item, apply the rule's Step 1 through Step 4 in order, for the
"<CLIENT_KEY>" client only. Output a JSON array, one object per item, in
the exact shape the rule's own "Output" section specifies — keep every
item's "id" field unchanged (these are blind ids like "s047"; do not guess
or invent the real id). Write the array to:
playbooks/tests/lead-rule/runs/<date>/judgments/<CLIENT_KEY>_<NN>.json

You are not told which items are traps or what group they belong to —
there is none of that information in the batch file. Judge each item on
its own text, exactly as the rule instructs.
```

Run all 60 in parallel where your environment allows it; each is
independent. Verify at the end that `runs/<date>/judgments/` has 60 files
(5 clients x 12 chunks) before scoring — a missing file scores as 10
missing judgments per `score.py`'s completeness warning, not a silent gap.

## 3. Score the run

```
python3 score.py \
  --dataset dataset.json \
  --run runs/<date>/judgments \
  --map runs/<date>/blind/map.json \
  --baseline baseline/baseline_metrics.json \
  --json runs/<date>/metrics.json
```

Read the printed report:
- **PASS/FAIL against targets**: lead >= 97%, tier >= 90%, groupA_warm >=
  95%, groupE_competitor >= 95%, groupFG_none >= 95%. These are the
  aspirational bar Owner Decision D set, not yet cleared by the approved
  baseline itself (the baseline scores lead 96.17%, just under the 97%
  target — see `README.md`). Treat this line as informational: a FAIL here
  alone does not block shipping.
- **PASS/FAIL against baseline**: none of those five may score *below* the
  baseline value on file. A rule edit that clears the fixed targets but
  drops group E (competitor) recall below baseline is still a regression —
  it does not ship. This is the binding line.
- **HOT diagnostics**: HOT precision, HOT recall, and non-HOT→HOT counts by
  expected decision, client, and group. They are informational against the
  checked-in v11 baseline, which intentionally has no HOT-specific fields.
  They become baseline-binding only if a future reviewed baseline explicitly
  includes `hot_precision`, `hot_recall`, or `nonhot_to_hot_rate`; the raw
  count/breakdown stay informational, and missing fields are never treated as
  zero.
- The confusion matrix and the list of failing rows (each with the trap it
  was testing and the judge's own `fit_reason`/`intent_reason`) tell you
  *why* it failed, not just that it did.

**Pass criteria for shipping a rule edit: the baseline PASS line must read
PASS.** The target PASS line is informational/aspirational until a rule
edit actually raises `lead_acc` past 97% — do not block a non-regressing
edit on it. If the baseline line fails, the edit is not done — fix the
rule (see "If it fails" below) and run again with a new `<date>` (or a
`-2` suffix on the same date).

## 4. Build the HTML report and attach it to the change

```
python3 build_report.py \
  --dataset dataset.json \
  --run runs/<date>/judgments \
  --clients clients.json \
  --rule ../../LEAD_QUALIFICATION_RULE.md \
  --map runs/<date>/blind/map.json \
  --baseline baseline/baseline_metrics.json \
  --out runs/<date>/report.html
```

Attach `runs/<date>/report.html` to whatever review or commit carries the
rule edit. It is a single self-contained file — open it directly in a
browser, no server needed. It shows every metric from step 3 plus a
filterable table of all 600 (scenario, client) rows: content, who the judge
thought the author was, expected vs. judged decision, the judge's own
reasoning, and a right/wrong flag. Filter by client, by group, "wrong
only", or free-text search.

## 5. If it fails

- Read the confusion matrix first — it tells you the *direction* of the
  drift (e.g. `warm->hot` means the rule is over-crediting routine posts as
  urgent).
- Read the failing rows for one over-represented trap group at a time,
  with their `trap` field and the judge's `fit_reason`/`intent_reason` —
  look for a pattern in *why* the judge got it wrong, not just that it did.
- Fix the **rule text** or the **buyer-profile format** (see README.md's
  "Why the buyer-profile format matters") — never fix a test scenario's
  `expected` label to match what the model produced. If you believe an
  `expected` label is itself wrong, that's a dispute — see "Adding
  scenarios" below; adjudicate it the same way, don't just change it.
- Have a fresh sub-agent (not the one that wrote the failing rule edit)
  make the rule fix, so the same blind spot doesn't just get rationalized
  away.
- Re-run from step 1 with a new date suffix. Do not reuse blind ids across
  runs meant for comparison — a fresh split is cheap and keeps the judge
  from ever seeing a real id.

## Adding scenarios

1. Write the new scenario(s) into `dataset.json` by hand, in the existing
   shape: `id, group, author_industry, surface, platform, community,
   language, author_line, text, date_note, trap` (the trap is a one-line
   note for humans reading the dataset, e.g. "same-trade competitor,
   different language" — it is stripped before any judge sees the item).
   Pick (or make) a group that names the trap this scenario tests.
2. Label it for **all five clients** in `clients.json`, each with
   `{"fit", "intent", "decision", "why"}` — reasoned from
   `playbooks/LEAD_QUALIFICATION_RULE.md`'s own steps and the client's
   buyer profile, never from intuition. Add a one-line `label_notes`
   summarizing all five decisions together, for humans skimming the file.
3. Use **two independent labelers** (two separate Sonnet sessions/contexts
   that do not see each other's work) to produce the five-client label set
   from steps 1-2 above. Where they disagree, a third session (the
   adjudicator — the coordinating agent, or a fresh sub-agent that has not
   labeled anything yet) reads both labelers' reasoning and the rule text,
   and decides; record its reasoning in `label_notes`. A group with a lot
   of disagreement usually means the rule itself is ambiguous there — fix
   the rule before you fix the label.
4. Re-run the full procedure (steps 1-4 above) with the enlarged dataset,
   and if the new baseline holds up (or improves) after a deliberate
   review, replace `baseline/judgments/` and `baseline/baseline_metrics.json`
   with the new run's output — see "Updating the baseline" below.

## Updating the baseline

Only do this deliberately, after a reviewed rule change that is meant to
become the new floor — not automatically after every passing run.

```
# from a completed runs/<date>/judgments/ that passed the baseline PASS line
# (the target line is informational until a rule edit clears 97% lead_acc):
rm -rf baseline/judgments
cp -r runs/<date>/judgments baseline/judgments
# baseline/judgments must use REAL ids, not blind ids — remap first if the
# run used --map (see the one-liner in this repo's git history for
# scripts/tests/lead-rule/baseline/judgments/*.json, or write real ids
# directly when scoring: score.py's --map only affects lookups, it never
# rewrites the judgment files on disk).
python3 score.py --dataset dataset.json --run baseline/judgments --json baseline/baseline_metrics.json
```

If your run's judgment files still carry blind ids, remap them to real ids
before copying into `baseline/judgments/` (baseline files are always keyed
by real dataset ids, so `score.py` runs against the baseline without
needing `--map`):

```python
import json, os
CLIENTS = ["solo_agency", "ai_video", "realtor", "insurance", "mlo"]
blind_map = json.load(open("runs/<date>/blind/map.json"))
for c in CLIENTS:
    rows = []
    for chunk_fp in sorted(__import__("glob").glob(f"runs/<date>/judgments/{c}_*.json")):
        rows += json.load(open(chunk_fp))
    for r in rows:
        r["id"] = blind_map.get(r["id"], r["id"])
    rows.sort(key=lambda r: r["id"])
    json.dump(rows, open(f"baseline/judgments/{c}.json", "w"), indent=2, ensure_ascii=False)
```

Commit `dataset.json` (if scenarios were added), `baseline/judgments/*`,
and `baseline/baseline_metrics.json` together with the rule change and the
attached `report.html` in one change, so the baseline and the rule it
describes never drift apart.

---

# Running the HOT-boundary canary (v12)

Use this small staged test when changing the Warm/HOT boundary. It isolates
whether the judge confuses “could benefit” or “is actively self-solving” with
observable demand to acquire external capability. It is domain-agnostic: each
case embeds a different `client_offer`/outcome and its own item evidence; the
scorer never contains occupations, industries, keywords, or provider names.

This workflow is intentionally separate from the 600-pair run above. For the
current v12 wording iteration, run the five-record canary below and stop; do
not launch the full 600-pair campaign and do not replace the v11 baseline. A
canary pass proves the targeted direction only—it is not a claim that the full
regression suite passed.

## 0. Files and binding contract

- `hot_boundary_cases.json` — 28 blind cases. Expected labels never appear in
  this file.
- `hot_boundary_expected.json` — fourteen Warm invariants, fourteen HOT
  positive flips, closed vocabulary, metamorphic pairs, and minimal flip
  relationships.
- `score_hot_boundary.py` — reads model judgments from one JSON/JSONL file or
  a directory of them.

The boundary run passes only when coverage/schema are complete, all binding
structural fields match, no non-HOT case becomes HOT, all positive flips stay
HOT, every emitted HOT satisfies the full v12 conjunction with substantive
evidence, explicit intent is paired only with explicit acquisition posture,
implied intent only with open posture, no generic evidence sentence is reused
across multiple Hot axes, and every fully present metamorphic pair is
structurally consistent.
`urgency`/`timing_evidence` are checked for vocabulary/evidence consistency but
never used as a HOT gate.

## 1. Stage exactly five blind records

Use this fixed canary set: two complete negative→positive flips (one
`explicit`, one `open`) plus one unrelated-provider negative.

```text
HB001,HB002,HB003,HB022,HB024
```

Create the file-in input under the run directory:

```python
import json
from pathlib import Path

ids = {"HB001", "HB002", "HB003", "HB022", "HB024"}
source = json.loads(Path("hot_boundary_cases.json").read_text(encoding="utf-8"))
canary = dict(source)
canary["cases"] = [row for row in source["cases"] if row["id"] in ids]
out = Path("runs/<date>/hot_boundary/canary_cases.json")
out.parent.mkdir(parents=True, exist_ok=True)
out.write_text(json.dumps(canary, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
assert len(canary["cases"]) == 5
```

## 2. Send the five records to one Luna extractor

Use `gpt-5.6-luna` in Codex (or the runtime's extractor tier), fresh context,
with this exact file-in/file-out brief:

```text
You are blind-judging exactly five lead items against a fixed rule. Read these
files in full before acting:

1. playbooks/LEAD_QUALIFICATION_RULE.md
2. playbooks/tests/lead-rule/runs/<date>/hot_boundary/canary_cases.json

For each case, judge ONE item for the client_offer embedded in that same case.
Use only that offer/outcome and that item's evidence. Do not infer buying
intent from an occupation, industry, launch, possible benefit, self-promotion,
peer advice, or active self-resolution. Apply the v12 steps in order.

Write exactly five JSONL objects, one per id and no prose, to:
playbooks/tests/lead-rule/runs/<date>/hot_boundary/canary_judgments.jsonl

Each object must contain these keys:
id, lead_rule_version, fit, fit_reason, problem_outcome,
problem_relevance, problem_state, resolution_activity,
acquisition_posture, intent, intent_reason, fit_evidence, problem_evidence,
unresolved_evidence, active_resolution_evidence, acquisition_evidence,
timing_evidence, urgency, counterfactual_result, relationship_type,
lead_level, decision, confidence.

Use the rule's closed vocabulary exactly. Evidence fields must name concrete
facts from the item; do not write an unsupported generic conclusion. Preserve
each id unchanged.
```

Before scoring, validate that the staged output contains exactly five unique
ids and no prose/fences. This is the TEAM_MODEL file-in/file-out canary gate;
the leader validates it and does not reclassify the five records inline.

## 3. Score the canary

```text
python3 score_hot_boundary.py \
  --cases hot_boundary_cases.json \
  --expected hot_boundary_expected.json \
  --judgments runs/<date>/hot_boundary/canary_judgments.jsonl \
  --ids HB001,HB002,HB003,HB022,HB024 \
  --json runs/<date>/hot_boundary/canary_metrics.json
```

The last line must be `BINDING PASS`. In particular, `non-HOT -> HOT` must be
`0/3`, HOT recall must be `2/2`, and contract/evidence/HOT-gate errors must all
be zero. A failure is evidence that the rule or output contract is still
ambiguous; do not relabel a fixture to match the model.

## 4. Optional 28-case boundary pass (still not the 600-pair run)

After the five-record canary passes and the rule text is stable, the remaining
23 cases may be judged in Luna files of at most five records each, then scored
together:

```text
python3 score_hot_boundary.py \
  --cases hot_boundary_cases.json \
  --expected hot_boundary_expected.json \
  --judgments runs/<date>/hot_boundary/judgments \
  --json runs/<date>/hot_boundary/metrics.json
```

This full boundary pass activates all fourteen metamorphic-pair checks. It is
still a 28-row focused test, not authorization to run or update the 600-row
baseline. Keep every judgment artifact under `runs/<date>/`; never copy these
files into `baseline/`.

---

# Running the comment-source regression test (Step 5)

Mandatory after any edit to Step 5 of `playbooks/LEAD_QUALIFICATION_RULE.md`
("Is this post a lead source in its comments?"). Do not ship a Step 5 edit
without a fresh run showing it does not score below
`baseline/comment_source/`. All paths below are relative to this directory.

## 0. Before you start

You need `playbooks/LEAD_QUALIFICATION_RULE.md` (Step 5 is the section
being scored — the earlier Steps 1-4 are not exercised by this test),
`clients.json` (the same 5 fictional buyer profiles used by the lead-rule
test — format only, never edited for a run), and `comment_posts.json` (69
posts, one line of text each, no comments attached — this test scores
whether the THREAD is a source, not any individual author).

## 1. Batch the posts (batch size 40 is not used here — one call per client)

Unlike the comment-triage test, `comment_posts.json` is small enough (69
posts) to judge in a handful of batches per client rather than one file per
post. Split it into chunks of ~12-15 posts (matching how
`baseline/comment_source/<client>_<n>.json` was produced) and write
`runs/<date>/comment_source_blind/<client>_<n>.json` — a plain array of
`{id, platform, community, author_line, text}` objects, `client` and
`secondary_client` fields never included (the judge reads one client's
profile per call, from `clients.json`, never both).

## 2. Spawn one Haiku sub-agent per (client x chunk)

Spawn with `model: "haiku"` — Step 5 is only proven if the model that
actually runs it in production (the harvest-gate classifier) is the one
judging it. Do not use a bigger model "to get cleaner numbers".

Exact prompt template (fill in `<CLIENT_KEY>` and the chunk file path):

```
You are blind-judging a batch of posts against a fixed rule's Step 5 only.
Read these two files in full before answering:

1. The rule: playbooks/LEAD_QUALIFICATION_RULE.md — apply ONLY "Step 5 —
   Is this post a lead source in its comments? (comment_source)". Ignore
   Steps 1-4 for this pass.
2. The client's buyer profile: the entry with "key": "<CLIENT_KEY>" in
   playbooks/tests/lead-rule/clients.json

Then read every item in this batch file:
playbooks/tests/lead-rule/runs/<date>/comment_source_blind/<CLIENT_KEY>_<n>.json

For each item, walk Step 5's five questions in order for the "<CLIENT_KEY>"
client only, and output a JSON array — one object per item, in the exact
shape Step 5's own "Output adds three fields" line specifies plus the
item's unchanged "id": {"id": "...", "types_match": "...", "comment_source":
"likely|unlikely", "comment_source_reason": "Q1 ... Q2 ... Q3 ... Q4 ...
Q5 ..."}. Write the array to:
playbooks/tests/lead-rule/runs/<date>/comment_source_judgments/<CLIENT_KEY>_<n>.json
```

Verify `runs/<date>/comment_source_judgments/` has one file per (client,
chunk) before scoring.

## 3. Score the run

```
python3 score_comment_source.py \
  comment_posts.json \
  comment_posts_expected.json \
  runs/<date>/comment_source_judgments \
  --show 20
```

Targets printed by the script: accuracy >= 93%, likely-recall >= 90%,
likely-precision >= 90%. The shipped baseline
(`baseline/comment_source/`, the v7 rule text now in Step 5) scores
**~87% accuracy, ~89% likely-recall** — below the aspirational targets but
the best measured round; see `README.md` for why it is still the adopted
text. Treat the target line as informational, the same way the lead-rule
test's target PASS line is: the binding check is that a rule edit must not
score *below* `baseline/comment_source/`'s numbers, printed by running:

```
python3 score_comment_source.py comment_posts.json comment_posts_expected.json baseline/comment_source
```

## 4. If it fails (scores below baseline)

Read the failing rows (`--show N`), each with the post text and the
judge's own `comment_source_reason` — the five questions are numbered so
you can see exactly which one the model got wrong. Fix Step 5's text, not
`comment_posts_expected.json`; if you believe an expected label is itself
wrong, treat that as a dispute the same way the lead-rule test does. Have a
fresh sub-agent make the fix. Re-run from step 1 with a new date.

## Updating the baseline

Only after a reviewed Step 5 edit meant to become the new floor:

```
rm -rf baseline/comment_source
cp -r runs/<date>/comment_source_judgments baseline/comment_source
python3 score_comment_source.py comment_posts.json comment_posts_expected.json baseline/comment_source
```

Commit `baseline/comment_source/*` together with the Step 5 edit.

---

# Running the comment-triage regression test (harvest job)

Mandatory after any edit to `playbooks/COMMENT_TRIAGE_RULE.md`. Do not ship
an edit without a fresh run showing it does not score below
`baseline/comment_triage/`. All paths below are relative to this
directory.

## 0. Before you start

You need `playbooks/COMMENT_TRIAGE_RULE.md` (the rule text being scored),
`clients.json` (same 5 fictional buyer profiles), and
`comment_threads.json` (24 comment threads, ~40 authors each, each thread
labeled for a `client` and a `secondary_client` — the same thread's authors
are judged twice, once per client, because the same author can be a lead
for one and a competitor for the other).

## 1. Use the blind thread files (batch size 40)

`comment_threads_blind/P01.json` .. `P24.json` are the blind batches
already used for the shipped v7 baseline — each is one thread's rows
(36-44 authors, code-prefiltered: no empty/emoji-only/"following"/"bump"/
tag-only rows), the same shape the production harvest job writes before
handing a batch to the classifier. Reuse them as-is for a like-for-like
comparison against `baseline/comment_triage/`, or regenerate fresh blind
batches from `comment_threads.json` (strip `trap`/`expected`, keep
`client`/`secondary_client`/`comment_source_reason` on the thread, `id`/
`author_line`/`comments` on each row) if the thread set has changed.

## 2. Spawn one Haiku sub-agent per (thread x client)

Each thread is judged once per labeled client (`client`, and
`secondary_client` when present) — up to 24 x 2 = 48 calls, matching
`baseline/comment_triage/`'s `{post}_{client}_0.json` /
`{post}_{client}_40.json` naming (a `_40` file appears only when a thread's
row count needed a second batch past 40; this test's threads all fit in
one batch of <=44, so expect mostly `_0` files). Spawn with `model:
"haiku"` — the classifier is the lowest model in production and must be
the one judged.

Exact prompt template (fill in `<CLIENT_KEY>`, `<POST_ID>`, and the thread
file path):

```
You are blind-judging one batch of comment-thread authors against a fixed
rule, for ONE client. Read these two files in full before answering:

1. The rule: playbooks/COMMENT_TRIAGE_RULE.md
2. The client's buyer profile: the entry with "key": "<CLIENT_KEY>" in
   playbooks/tests/lead-rule/clients.json

Then read the source post and rows in this batch file:
playbooks/tests/lead-rule/comment_threads_blind/<POST_ID>.json

Apply the rule's Step 1 through Step 3 to every row in "rows", for the
"<CLIENT_KEY>" client only. Judge the AUTHOR, not the comment's politeness
or topic. Answer every row by id; never skip one. Output a JSON array in
the exact shape the rule's own "Output" section specifies, id unchanged.
Write the array to:
playbooks/tests/lead-rule/runs/<date>/comment_triage_judgments/<POST_ID>_<CLIENT_KEY>_0.json
```

Verify the judgments directory has one file per (thread, client) pair
judged before scoring.

## 3. Score the run

```
python3 score_comments.py comment_threads.json runs/<date>/comment_triage_judgments --show 20
```

Targets printed: accuracy >= 93%, keep-recall >= 95%, keep-precision >=
90%, competitor-drop >= 95%, noise-drop >= 95%. The shipped baseline
(`baseline/comment_triage/`, the v2 rule round — see `README.md` for why
v2's judgments and not v7's own are the kept baseline) scores **~94.5%
accuracy, ~92.5% keep-recall**; reproduce with:

```
python3 score_comments.py comment_threads.json baseline/comment_triage
```

The binding check is the same as elsewhere in this file: a rule edit must
not score *below* those baseline numbers, even if it clears (or misses)
the informational targets above.

## 4. If it fails (scores below baseline)

Read the confusion matrix and the failing rows (`--show N`, each with the
author line, their comments, and the judge's own `reason`). Fix
`playbooks/COMMENT_TRIAGE_RULE.md`'s text, never `comment_threads.json`'s
`expected` block. Have a fresh sub-agent make the fix. Re-run from step 1
with a new date.

## Updating the baseline

Only after a reviewed rule edit meant to become the new floor:

```
rm -rf baseline/comment_triage
cp -r runs/<date>/comment_triage_judgments baseline/comment_triage
python3 score_comments.py comment_threads.json baseline/comment_triage
```

Commit `baseline/comment_triage/*` together with the rule edit.
