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
