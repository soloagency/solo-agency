# Content Signal Research

Load this playbook only for `research_content_signals`, or for an explicitly requested combined
research-then-writing flow. Daily Run does not load it and never opens comment threads.

## Purpose and boundary

Find reusable audience questions, pains, objections and real wording in a small number of recent
notable posts. Write compressed, source-traceable signals to the Content Evidence Bank. Do not
write a script, blog, caption, CTA, CRM contact, lead record or outreach message.

This is not `harvest_thread`. `harvest_thread` remains a one-source, Boss-ordered CRM workflow and
is the only workflow permitted to capture commenters as leads or change a discovered source to
`harvested`.

## Inputs and selection

Read the current Client Intelligence Profile, recent post-level evidence, recent Idea Matrix and
the complete `research/content_signals_snapshot.json` when it exists. Rank recent candidates by:

1. audience and problem/outcome relevance;
2. post-level evidence of a question, disagreement, failed attempt or decision tension;
3. meaningful comment count and the likelihood that replies add detail;
4. freshness and novelty against the active evidence snapshot; and
5. a plausible, value-first lesson, rather than a promotional topic.

Use recent `likely` discovered threads as the first candidate pool. Other recent post-level
evidence is eligible when its URL, platform and reason are recorded. Do not treat a high comment
count as quality by itself. Select at most three threads per run. Skip a thread already researched
inside its freshness window unless its post or comment count materially changed.

## Fixed collection budget

- Start with at most 40 meaningful comments per selected thread after collector-side removal of
  empty, emoji-only, tag-only, bump and obvious spam rows.
- Hard-cap the entire task at 120 meaningful comments, including every later page.
- Fetch a later page only when the first page has high signal density: at least six eligible
  signals or 15% of meaningful rows form usable, non-duplicate signal candidates.
- Stop deeper pagination when density falls below that threshold, a rate/safety warning appears,
  the remaining global cap is zero, or no new signal type/problem emerges.

The task is serial and read-only. Respect the platform's existing request and pacing limits. A
platform error, login warning or safety trip ends collection for that thread and is recorded as a
partial run. It never causes the task to broaden to more threads.

## Extract, validate and write

1. Stage comments in files of no more than 40 rows. Preserve platform IDs and source URLs in the
   staged evidence, but do not put raw text or PII in the model-routing log.
2. If more than five comments need classification, route the extraction to Luna. First run a
   five-row canary, validate exact IDs, schema and closed vocabulary, then process the remaining
   batches. A failed Luna batch may receive one logged Terra retry. If no extractor is available,
   checkpoint with `low_tier_subagent_unavailable`; never classify a bulk set inline.
3. Luna classifies only into `question|pain|complaint|objection|misconception|failed_attempt|
   desired_outcome|buying_signal`, normalizes audience/problem/outcome, identifies duplicates and
   returns the shortest necessary evidence excerpt. The Team Leader validates samples, provenance,
   budgets and schema; it does not reclassify the full batch.
4. Write source-traceable post and comment signals to the append-only Content Evidence Bank schema
   in `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`. Keep CRM and PII out of the bank. A duplicate
   appends a dedupe revision using the same `signal_id`, never a second unrelated signal.
5. Build the next `content_signals_snapshot.json` in a temporary sibling, validate every row has
   `evidence_excerpt`, source URL, signal type, confidence, freshness and dedupe hash, then rename
   atomically. On failure retain the prior snapshot.
6. Append one run summary to `content_signal_runs.jsonl`: task/run IDs, selected-thread count,
   meaningful-comment count, pages read, signals added/revised, skipped duplicates, outcome and
   non-sensitive blocker code when applicable.

## Output to Idea Matrix

The next Daily Run consumes only the complete, non-expired snapshot. It may combine post-level and
comment signals, but each matrix item must cite `evidence_ids`, state a specific `viewer_lesson`,
show a non-promotional angle, confidence and freshness, and pass the Audience Value-First Gate.
One comment stays low-confidence evidence. Repeated independent wording can raise evidence count
and confidence, never fabricate consensus.

The report shows a compact insight, count, safe representative excerpt(s) and source link(s). It
does not print a raw comment dump. If the snapshot is missing, expired or partial, report `comment
evidence unavailable` and continue from post-level evidence; do not open comments during Daily
Run.

## Scheduling and combined requests

This task is on demand by default. It can run before a production request or as a separately
approved automation scheduled before Daily Run. It must not overlap the same client's Daily Run.
An optional combined request may run this research first, present/select the resulting idea, then
invoke the separate writing task with the selected `idea_id`. The research phase never writes the
content itself.
