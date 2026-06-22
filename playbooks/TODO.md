# Solo Agency TODO

This TODO sits beside the detailed child playbooks as an optimization backlog. Do not treat these notes as mandatory daily questions for the human.

The goal is fully automatic daily marketing operation where the human spends only a few minutes approving, correcting, or blocking sensitive actions.

## Daily UX Principle

- New layers must become automatic memory, scoring, and decision logic.
- They must not become a daily questionnaire.
- The daily report should be approval-ready.
- The human should usually decide in 3-7 minutes: approve, revise, choose another option, skip, publish, or ask for production.

## Future Improvement Backlog

### Owner / Founder Marketing Profile

Add a one-time or progressive profile for the owner using the system for themselves:

- current offer and secondary offers;
- 30-day and 90-day business goals;
- ideal customers;
- customers the owner does not want to serve;
- preferred channels;
- personal brand voice;
- CTA / booking link / lead magnet;
- time available per day for approval, recording, publishing, and lead review.

Rule: infer first, ask only for corrections. Do not ask this every day.

### Offer And Funnel Map

Add a conversion map so content is tied to business outcomes:

- stranger sees content;
- viewer comments, clicks, replies, or books;
- lead is classified;
- human approves outreach if needed;
- follow-up is tracked;
- sale/booking outcome feeds learning.

Track which content assets create leads, not only views.

### Daily Marketer Brief

Before picking the daily idea, the agent should load:

- active goal;
- active campaign;
- yesterday's published content status;
- yesterday and 7-day metrics;
- open leads not handled yet;
- current blockers;
- today's production constraints.

This should be internal context unless the human needs to approve something.

### Campaign Layer

Add a 7-day and 30-day campaign layer:

- content series;
- launch sequence;
- nurture sequence;
- weekly theme;
- campaign objective;
- daily assets as pieces of one larger strategy.

The agent should avoid random one-off ideas when a campaign is active.

### Audience Objection Bank

Build automatically from comments, leads, private groups, sales calls, and competitor content:

- repeated questions;
- objections;
- fears;
- misconceptions;
- comparison points;
- exact audience language;
- proof gaps.

Use the bank to improve hooks, scripts, CTAs, and lead-gen angles.

### Creative Quality Rubric

Before showing a draft, score it internally:

- hook specificity;
- pain-point fit;
- evidence strength;
- CTA clarity;
- differentiation from competitors;
- urgency/timeliness;
- production readiness;
- lead-generation potential.

Only show the score summary when it helps the human decide.

### Lead Handling Protocol

Expand lead workflow:

- hot / warm / cold classification;
- suggested response draft;
- approval before any outreach;
- follow-up after 1, 3, and 7 days;
- status: new, reviewed, contacted, booked, won, lost, do_not_contact;
- learning from won/lost leads.

### Personal Content Asset Bank

Store reusable owner-specific assets:

- personal stories;
- case studies;
- demos;
- testimonials;
- screenshots;
- product explanations;
- strong opinions;
- approved phrases;
- banned angles.

Use approved assets before inventing new creative directions.

## Testing And Optimization Loop

After each test run with an AI agent:

1. Save the full conversation between the human and the agent into a log file.
2. Put the log under a review folder such as:

```text
daily-content-pipeline/test_logs/YYYY-MM/
```

3. Ask Codex to read the log and review:
   - where the agent skipped a required stage;
   - where it asked unnecessary questions;
   - where it jumped to schedule, production, or private setup too early;
   - where it failed to load a needed child playbook;
   - where it produced Markdown instead of HTML;
   - where it missed approval gates;
   - where it treated WideCast as the identity instead of one provider path;
   - where it failed to measure yesterday / 7-day results;
   - where the daily UX took too much human time.
4. Update the relevant child playbook with the smallest rule/gate needed to prevent that miss.
5. Re-test.

## Current Priority

First priority is not adding these advanced layers. First priority is validating the modular split:

- Does the agent load Stage 0 + Stage 1 before setup?
- Does it avoid asking for industry/sub-industry?
- Does it run the first public report before private collector setup?
- Does it ask private source and production questions at the right time?
- Does it load Stage 2/8 before private-source work?
- Does it load Stage 3 before production/provider work?
- Does it load Stage 5 after publishing?
- Does it pass Stage 9 self-audit before claiming completion?
