# Comments In Groups — the `comment` campaign channel

Stage: `18`

## Load Rule

Load whenever a client has a campaign with `channel_strategy: comment` — during Setup Flow when
one is created or edited, and in EVERY scheduled run's step for collector-executed channels
(AUTOMATION_SCHEDULING §H). LOAD LEDGER applies.

## What this channel is

Email starts from a PERSON: pick a contact out of the pipeline, enrich them, write to their
address. **A comment runs the other way round.** It starts from a POST inside a group the
operator named, and the thing being addressed is that post — not anybody's inbox.

That difference is the whole stage, and getting it backwards is the failure this file exists to
prevent: on 2026-08-15 an agent, finding no path for this channel, hand-wrote three draft files
straight into `outbox/pending_approval/` with invented ids, no budget check and no dedupe. Every
gate below was missing on that route.

- **Operator ruling 2026-08-17 — a commented author NEVER becomes a CRM contact.** Commenting is
  a visibility play, not lead capture. The draft carries no `lead_id`; nothing is written to the
  CRM. If that person later replies to the comment, THAT is when they become a contact — a reply
  is interest, a post is not.
- **Operator ruling 2026-08-17 — drafting follows PUBLISH CAPACITY, not `daily_quota`.** A
  campaign with one group and `comments_per_group_per_day: 1` can publish one comment a day. Its
  "Daily draft budget" field is a drafting bound for EMAIL and is not read here. `draft comment`
  refuses past `capacity × 2 days` with `capacity_horizon_full`, so the operator is never asked
  for forty decisions to ship one comment.
- **Approval is the command.** The operator approving on the Approvals page IS the instruction to
  publish; there is no second "send it" step for him to remember.

## Every scheduled run — what the agent does

### 1. Load what you are allowed to say

Four inputs. A comment written from fewer than all four is the one that reads like anyone could
have left it:

1. **The client profile** (`clients/{slug}/.../client_profile_*.md`) — who this client is, what
   they do, what they have actually done. This is the material you may draw on. The rule below
   ("no claim the client's own experience cannot support") tells you what you must NOT say; this
   is the step that tells you what you MAY, and without it the agent improvises.
2. **`goal.description`** (`campaign get --slug X`) — the operator's own words about who is worth
   answering and what to say. It is the ONLY criterion for whether a post deserves a reply. Do not
   substitute a generic "be helpful" instinct for what he wrote, and do not widen it because a
   post is interesting.
3. **The group's name**, from `items[].group.name` in the scan — the url tells you nothing, while
   "Help for Insurance Agents" tells you the field, that the readers are AGENTS rather than
   customers, and therefore the register: a peer talking to peers. Pass it as `group_name` on the
   draft.
4. **The post itself** — see steps 2 and 3.

`goal.message_bank` and `goal.cta` are deliberately NOT read on this channel (operator ruling
2026-08-17) — they belong to email, and the campaign page hides them here.

### 2. Scan the campaign's groups

For each url in `audience.groups`, enqueue ONE read job:

```json
{ "name": "scan <group> for comment targets",
  "url": "<group_url>",
  "platform": "facebook",
  "capability": "fb.group.posts",
  "inputs": { "group_url": "<group_url>", "within_days": 2, "max_pages": 2 } }
```

`within_days: 2` is deliberate: a comment on a four-day-old post is invisible, and the post may
be gone by the time it is approved. Never scan without a time bound.

Groups are the operator's list. Never comment in a group that is not in `audience.groups`, and
never add one yourself — `draft comment` refuses it (`group_not_in_campaign`) and that refusal is
correct, not an obstacle to route around.


**Pin the scan to ONE collector account, and remember which.** Enqueue the read with
`allowed_extension_instance_ids: ["<instance_id>"]`, chosen from the extensions the bridge lists
on `/status`. That account is the only one PROVEN to be a member of the group — Facebook serves a
non-member the post with no composer at all, which is exactly the failure measured on 2026-08-17:
the collector read the post perfectly and then reported `comment composer not found`. Pass the same
id as `"collector"` when you deposit the draft; the bridge publishes from that account and from no
other, and an item whose account is not checked in waits rather than going out under a different
name.

### 3. Filter first, THEN judge

```
tool crm-store --client-dir {outreach} draft judged --campaign X
```

returns every post this campaign has already decided about — drafted, posted or skipped — by
canonical post id. **Drop those from the scan before you read a word of them.** A scan with
`within_days: 2` returns the same posts run after run, so without this step yesterday's thirty
skips are re-judged today, and the day after: the same post gets read three or four times before
it falls out of the window.

The waste is the smaller half. The real cost is that re-judging is **unstable** — the same post
skipped today can be drafted tomorrow with nothing new behind the change, which quietly makes
"the goal is the only criterion" untrue in practice. A judgement is taken once.

**If the filtered list is empty, that group is done for now — move to the next group.** Do not
re-read it, do not lower the bar to find something. A group with no new posts is a normal
outcome, not a problem to solve.

Then, for every post that survives the filter, answer in order and stop at the first "no":

1. Does the goal actually cover this person's situation? A post outside the brief is skipped even
   when a good answer is obvious.
2. Can you say something that is **useful on its own** — a thing you know from doing this work
   that the reader does not? A comment that only signals presence ("great post!", "DM me") is
   worse than no comment: it spends the account's one daily action and buys nothing.
3. Is there a real question or a real problem stated? Answer that, not the topic.
4. (Already handled by the filter above — the code still refuses a repeat, with
   `already_drafted` or `already_judged` naming which decision it was.)

Skipping is the normal outcome. A scan that yields two comments out of thirty posts is a good
scan.

### 4. Write the comment, per the goal's instructions

- Answer the post's actual question. Every claim must be something the client's own experience
  supports; nothing invented, no statistics that are not in the client profile.
- No link, no pitch, no "message me" in a first comment. The profile is the call to action.
- The reader is a stranger in a public group: write to the room, not to one person.
- Length is whatever the answer needs and no more.

### 5. Deposit it in the approval queue

```
tool crm-store --client-dir {outreach} draft comment --campaign X --json '{
  "post_url": "<permalink>", "group_url": "<group url from audience.groups>",
  "post_author": "<display name as shown>", "post_excerpt": "<first ~200 chars of the post>",
  "body_text": "<the comment>", "post_seen_at": "<ISO time the scan observed it>",
  "group_name": "<items[].group.name from the scan>",
  "collector": "<the extension instance id that read this group>" }'
```

**Record the skips in the same pass.** Everything you read and decided not to answer goes back
in one batch, or it returns tomorrow for another judgement:

```
tool crm-store --client-dir {outreach} draft skip --campaign X --json '{
  "group_url": "<group url>",
  "posts": [ {"post_url": "<permalink>", "reason": "<one short line>"},
             {"post_url": "<permalink>", "reason": "<one short line>"} ] }'
```

A skip is permanent, on purpose: a decision that was worth taking is worth keeping. The reason is
what makes it auditable later — "outside the goal", "no real question", "already answered by
three people" are all fine; an empty reason is not.

This is the ONLY way a comment draft may enter the system. Never write a file into
`outbox/pending_approval/` yourself — the command mints a real id, checks the group, pins the
post, refuses a duplicate and enforces the capacity ceiling, and a hand-written file has none of
that. `tool crm-store draft capacity --campaign X` tells you how many more will be accepted.

Refusals are answers, not errors to retry: `group_not_in_campaign`, `post_outside_group`,
`unpinnable_post`, `already_drafted`, `capacity_horizon_full`, `campaign_paused`,
`channel_mismatch`, `no_capacity`. Report the count and reason; do not work around them.

### 6. Publishing is NOT the agent's step — the bridge does it

The operator approves on `/ui/{client}/approvals`, and **that approval publishes the comment**.
The bridge applies the decision in-process, moves the draft to `outbox/approved/` with
`decided_by: ui`, and enqueues `fb.post.comment` itself (`comment_dispatch.go`). Nothing waits for
a run. The agent **never** enqueues `fb.post.comment` for a campaign of this channel — not to
"help", not to retry, not when the operator asks for the day's work to be finished.

What the bridge guarantees, so you do not need to reason about it:

- **One at a time, spaced.** The first approval publishes at once; the ones behind it come due at
  least 25 minutes apart. Approving six drafts in one sitting does not produce six comments in a
  minute.
- **The daily caps are read.** `comments_per_group_per_day` and
  `comment_groups_per_account_per_day` (`/ui/settings`) are enforced at approval; a capped
  approval is not lost, it comes due after the local-day rollover and the operator is told so on
  the spot.
- **The outcome is recorded.** A record with `status: done` and `verified: true` marks the draft
  `sent` and the post `posted` in `commented_posts.json`. Anything else marks the draft `blocked`
  with the collector's own reason **and releases the post**, so a later scan may try it again — a
  comment that never landed must not permanently burn its target.
- **Target freshness is deliberately not checked** (operator ruling 2026-08-17). A post that has
  aged is still commented on if the operator approves it. If that changes, it changes in code.

So the run's only remaining duty for this channel is the compose loop above, and reporting.

## The run reply — one line per comment campaign

> `{campaign} — scanned {N} posts in {G} group(s), {S} already judged and skipped over, {K} worth
> answering, {D} drafted ({R} refused: reason), {A} awaiting your approval, capacity {C}/day.`

Say the refusal reasons out loud. A run that drafts nothing because the ceiling is full is a
healthy run and must say so; a run that drafts nothing because every scan failed is not, and the
difference has to be visible without opening a terminal.

## Storage (Stage 7 addendum)

- `campaigns/{slug}/outbox/pending_approval/{day}/draft_*.json` — the draft; `channel: comment`,
  `lead_id: ""`, `to` = the post permalink, plus `post_url`, `group_url`, `post_author`,
  `post_excerpt`, `post_seen_at`.
- `campaigns/{slug}/commented_posts.json` — one entry per post ever drafted for, under flock. It
  outlives the draft (which leaves the outbox once published), and it is what makes "one comment
  per post, ever" true rather than hopeful.
- Campaign config keys read here: `audience.groups`, `goal.description`. `daily_quota` and
  `sendboxes` are NOT read on this channel and are hidden on the campaign page.

## Worked shape (fictional — never copy the specifics)

Goal says: *"answer people setting up their first commercial kitchen, so cooks in the city
recognise the name."* The scan returns a post: *"third quote for the hood system and they're all
different, what am I missing?"* → in scope, has a real question, and there is something concrete
to teach → comment names the three line items quotes usually differ on and why. A second post in
the same group asks for restaurant recommendations → skipped: outside the goal, no matter how
easy a reply would be.

The industry above is fictional on purpose. If a real client's industry ever appears in this
file's examples, it will be copied into live output — that has happened four times.
