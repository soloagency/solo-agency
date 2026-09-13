# Revenue Engine reply acceptance — manual contract test

Use this checklist after any edit to the Team Leader Reply Frame, Human Action Highlighting
Contract, Feature Discovery Rule, Feature Catalog, or Setup-Complete Closing Template. Test in the
Boss's language. A single failed invariant is a release blocker because optional feature discovery
must never obscure a required human action.

## Case 1 — normal post-setup reply

Prompt: ask Sam a harmless information question for a client when no human action is pending.

- [ ] The reply answers the question first.
- [ ] It contains exactly one compact Revenue Engine awareness anchor.
- [ ] The anchor uses `/ui/{client_slug}` when a client is in context, `/ui` when none is in context,
      or the phrase `show me the features` when the local dashboard is unavailable.
- [ ] The anchor appears after any contextual feature-tour block and before next jobs.
- [ ] The reply ends with the next-jobs block and exactly one closing question; nothing follows it.
- [ ] The anchor does not auto-navigate the dashboard.

## Case 2 — required human action

Prompt: create a real blocker that requires the Boss to approve, connect, paste, run, click, or
answer one question.

- [ ] The reply contains the correct standalone `**[ACTION REQUIRED]**` block.
- [ ] It contains no Revenue Engine awareness anchor.
- [ ] It contains no optional feature-tour block and no next-jobs offers.
- [ ] The most important required-action block is the final block; no optional link, prompt, CTA, or
      question follows it.
- [ ] Feature awareness returns only in the next reply after the required action is resolved.

## Case 3 — complete feature-menu request

Prompt: `What can Solo Agency do?` or an equivalent everyday phrase.

- [ ] Sam shows all 25 jobs in the four canonical Feature Catalog groups.
- [ ] Sam opens or prints the current client's `/ui/{client_slug}` Overview route.
- [ ] The compact awareness anchor is omitted because the complete menu already provides it.
- [ ] Sam asks which job the Boss wants, without inventing a second feature list.

## Case 4 — protected surfaces

Check each surface separately: Setup First Words, Setup Flow before completion, provider/scheduled
notification, INTERNAL_REPORT, and a client-facing artifact.

- [ ] No surface contains the Revenue Engine awareness anchor.
- [ ] First Words remain the exact Team Leader introduction with no prefix or suffix.
- [ ] Notifications and artifacts keep their existing copy contracts and do not gain optional
      dashboard links or feature prompts.

## Case 5 — setup completion branches

Healthy branch: first-run dispatch succeeded and no action is pending.

- [ ] Order is current run state → Revenue Engine anchor → three offers → Standing Invitation → one
      closing question.

Failure branch: dispatch is genuinely impossible and the Boss must act.

- [ ] Order is failure context → `**[ACTION REQUIRED]**` as the exclusive close.
- [ ] The three offers, Revenue Engine anchor, Standing Invitation, feature tour, and closing
      next-jobs question are all absent and deferred.

## Recording

Record one line per case:

```text
case {n} · {runtime} · pass|fail · exact misplaced/missing element
```
