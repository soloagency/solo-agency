# Lead qualification rule — regression test

This directory is the permanent, in-repo regression test for
`playbooks/LEAD_QUALIFICATION_RULE.md` — the Fit x Intent rule every
classifier of posts, comments, captions and people rows applies (Stage 10
detection, the feed/surface pass, people search, the lead-engine skill's
extractor tier). Any edit to that rule file must be re-scored against this
test before it ships. See `RUN.md` for the exact procedure.

## What's here

| File | What |
|---|---|
| `dataset.json` | 120 hand-built scenarios (posts, comments, captions, bio-only profile rows), each scored against all 5 example clients below = 600 (scenario, client) judgments per run. Grouped A-I by what trap each group tests (see "Groups" below). Each scenario carries a final `expected` decision per client, reasoned from the rule text and the client's buyer profile — not from intuition. |
| `clients.json` | 5 **fictional** example buyer profiles in the `buyer_profile` format the rule reads: `sells`, `sells_to`, `types` (category first, examples after "e.g."), `why_they_need`, `location`, `competitors`, `not_buyers`. Used as *examples* of the format, never copied into a real client's profile. |
| `score.py` | Scores a directory of judgments against `dataset.json`. Pure Python 3, no dependencies. Prints binary (lead/non-lead) and tier (5-way decision) accuracy, per-client and per-group breakdowns, three trap-group recall metrics, a confusion matrix, and failing rows with the judge's own reasons. Exits 1 if below the target thresholds or below a given baseline. |
| `build_report.py` | Builds a self-contained HTML report from the same data (metrics + a filterable row-by-row table + the 5 client profiles + the rule text scored). Imports `score.py` for the metrics so the two never disagree. |
| `split_blind.py` | Splits `dataset.json` into blind chunks (10 scenarios each, group/trap/expected/label_notes stripped, ids reassigned as `s001`, `s002`, ...) plus a `map.json` back to real ids — what a judge actually reads. |
| `baseline/judgments/<client>.json` | The round-11 judgments from the original 11-round campaign that produced `rule_v11.md` (now `playbooks/LEAD_QUALIFICATION_RULE.md`), remapped from blind to real scenario ids. 5 files, 120 judgments each, 600 total. |
| `baseline/baseline_metrics.json` | `score.py`'s output on the baseline judgments — the floor every future run must clear. |
| `RUN.md` | Step-by-step procedure for running this test again after any rule edit, and for adding new scenarios. |

## The numbers this test exists to protect

The round-11 campaign (120 scenarios x 5 clients = 600 blind judgments by
Haiku, the model that actually runs this rule in the product) scored:

- **96.2% binary lead/non-lead accuracy** (target: >= 97%, not yet cleared)
- **93.8% tier accuracy** — hot/warm/watch/none/competitor exactly right (target: >= 90%, cleared)
- **96.0% group A "right kind of person, no stated need" recall** — the single most important trap: a real prospect who hasn't said anything yet must be kept as `warm`, never dropped (target: >= 95%, cleared)
- **100% group E "competitor in disguise" recall** — someone selling the same thing to the same people, however friendly the post reads, must never become a lead (target: >= 95%, cleared)
- **97.8% group F/G "noise / negation" recall** — memes, job seekers, spam, and explicit "not looking" must resolve to `none` (target: >= 95%, cleared)

Run `python3 score.py --dataset dataset.json --run baseline/judgments` in
this directory to reproduce all five numbers exactly.

## Why the buyer-profile format matters

The single biggest accuracy gain in the 11-round campaign (round 7 to round
8, ~88% to 93-96% binary accuracy) came from rewriting the buyer-profile
*input contract*, not from rewording the rule itself. Two format rules carry
almost all of that gain, and any new client profile — real or a future test
fixture — must keep them:

1. **Category first, examples after "e.g."** — `types` lines read
   "independent professional of ANY trade who markets themselves, e.g.
   real-estate agent, loan officer, dentist..." never a bare list of trades.
   A small model tested against a bare list starts requiring an exact
   dictionary match; tested against "ANY trade, e.g. ..." it generalizes
   correctly to a dentist or an escrow officer it has never seen named.
2. **Describe who they are, never an activity the product enables.**
   "People who make videos" makes a small model demand proof that the
   person already films before it will call them a fit. "Independent
   professional who markets themselves" doesn't — someone who has posted
   zero video is still squarely a fit, and the *lack* of video is exactly
   the gap `why_they_need` names.

`clients.json`'s own `format_note` field states this rule; every one of its
5 profiles follows it.

---

## Comment-source test (Step 5 of the lead rule)

Regression test for `playbooks/LEAD_QUALIFICATION_RULE.md` Step 5 ("Is
this post a lead source in its comments?") — the question that decides
whether a post gets recorded as a `discovered` source in the client's
source registry. This test never harvests anything; it only scores
`likely`/`unlikely` against the 5 example clients in `clients.json`. See
`RUN.md`'s "Running the comment-source regression test" section for the
procedure.

| File | What |
|---|---|
| `comment_posts.json` | 69 hand-built posts (platform, community, author line if any, text) — a mix of requests, peer/gathering questions, and promotions, across the same 5 clients' industries plus out-of-industry posts to test generality. No comments attached: this test scores the THREAD, not any individual author. |
| `comment_posts_expected.json` | Keyed by post id -> `{client_key: "likely"\|"unlikely"}`, one entry per (post, client) pair scored, reasoned from Step 5's own five questions against each client's `clients.json` profile. |
| `score_comment_source.py` | Scores a judgments directory against `comment_posts.json` + `comment_posts_expected.json`. Accepts paths relative to this directory (or absolute). Prints accuracy, likely-recall, likely-precision, a per-client breakdown, and failing rows with post text + the judge's own `comment_source_reason`. Exits 1 below target or baseline. |
| `baseline/comment_source/<client>_<n>.json` | The v7-round judgments (the same rule text now shipped as Step 5), 30 files across the 5 clients. |

**Numbers this test protects:** run
`python3 score_comment_source.py comment_posts.json comment_posts_expected.json baseline/comment_source`
to reproduce — **~87.2% accuracy, ~89.0% likely-recall, ~81.8%
likely-precision** (345 (post, client) judgments, 91 expected `likely`).
These sit below the aspirational targets in `score_comment_source.py`
(accuracy/recall/precision >= 90-93%) — Step 5 asks a harder, more
open-ended question than the main lead rule (every reply chain a post
*could* produce, not one item's own fit), and this was the best round
measured across 8 iterations of the rule text (v1-v8). The binding check
for a future edit is the same pattern as the main test: do not ship a
Step 5 change that scores below these numbers, even though they are
themselves below the target line. Nothing is harvested off a `likely`
verdict automatically — the Boss approves every discovered source by hand
(see `playbooks/LEAD_QUALIFICATION_RULE.md` Step 5 and Owner Decision 1,
2026-09-11) — which is why a ~87%/~89% classifier is an acceptable gate,
not a shipped decision.

## Comment-triage test (the harvest job's classifier)

Regression test for `playbooks/COMMENT_TRIAGE_RULE.md` — the batch
classifier that decides keep/drop for each author in an *approved*
discovered thread, run only when the Boss orders a harvest (never in the
daily run). See `RUN.md`'s "Running the comment-triage regression test"
section for the procedure.

| File | What |
|---|---|
| `comment_threads.json` | 24 labeled comment threads (a source post + ~40 comment-author rows each, code-prefiltered — no empty/emoji-only/"following"/"bump"/tag-only rows), each thread scored against a `client` and (where relevant) a `secondary_client`, mirroring how the same author can be a lead for one client and a competitor for another. Each row carries a final `expected` block (`fit`, `competitor`, `noise`, `decision`, `why`) per labeled client, reasoned from the rule text — not intuition. |
| `comment_threads_blind/P01.json` .. `P24.json` | The blind batches actually judged to produce the shipped baseline — one thread's rows per file (36-44 authors after prefilter), `trap`/`expected` stripped, the same shape the harvest job hands the classifier in production. |
| `score_comments.py` | Scores a judgments directory against `comment_threads.json`. Accepts paths relative to this directory (or absolute). Prints accuracy, keep-recall/precision, competitor-drop, noise-drop, a per-client breakdown, a confusion matrix, and failing rows with the author line, their comments, and the judge's own `reason`. Exits 1 below target or baseline. |
| `baseline/comment_triage/<post>_<client>_<n>.json` | The kept baseline judgments, 44 files. |

**Numbers this test protects:** run
`python3 score_comments.py comment_threads.json baseline/comment_triage` to
reproduce — **~94.5% accuracy, ~92.5% keep-recall, ~97.2%
keep-precision, ~98.3% competitor-drop, ~99.6% noise-drop** (1427 (row,
client) judgments, 7 missing).

**Why the baseline is the v2 round, not the shipped v7 text.** The rule
text actually shipped as `playbooks/COMMENT_TRIAGE_RULE.md` is v7 — it
adds the `stated_trade`/`author_product` fields and a stricter two-part
competitor test (owner-required, see Owner Decision 3, 2026-09-11) that
v2 lacks. But the *judgments* kept in `baseline/comment_triage/` are from
the v2 round, the highest-scoring round measured across the full
iteration (iter1-iter7): v2 scored the numbers above, while the v7 rule's
own round measured **89.6% accuracy** — on a run that included one
corrupted batch file, so that number understates v7's true score somewhat.
v7 was still adopted as the shipped rule text for its stricter competitor
test and the two new fields, which matter more for CRM data quality than
the small measured accuracy gap. Because `score_comments.py` compares
judgments against `comment_threads.json`'s fixed `expected` labels
regardless of which rule text produced them, keeping the higher-scoring
v2 judgments as the numeric floor is still a valid regression gate for any
future edit to the v7 text: an edit must keep clearing the v2 numbers, not
just match v7's own.

## Groups in `dataset.json`

| Group | Size | Tests |
|---|---|---|
| A | 30 | Right kind of person, **no stated need** — must stay `warm`, never dropped |
| B | 15 | Multi-intent items (asks for two different kinds of provider in one post) |
| C | 15 | Out-of-area / adjacent-profession edge cases |
| D | 15 | Vague fit with thin signal — must still resolve per-client |
| E | 12 | **Competitor in disguise** — same trade, same market, must never become a lead |
| F | 12 | Pure noise — memes, MLM, job seekers, hiring posts, spam |
| G | 8 | Negation / already-resolved — explicit "not looking", stale, already fixed |
| H | 8 | Bio-only rows, occupation-only headlines, zero body text |
| I | 5 | Cross-cutting scenarios that split differently across all 5 clients from one post |

## Language and surface mix

English, Vietnamese with diacritics, Vietnamese without diacritics, and
mixed-language code-switching, matched to the surfaces the rule actually
reads in production: group posts, comments, captions, and bio-only profile
rows.
