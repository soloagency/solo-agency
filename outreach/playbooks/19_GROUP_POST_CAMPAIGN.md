# Posts Into Groups — the `post` campaign channel

Stage: `19`

## Load Rule

Load whenever a client has a campaign with `channel_strategy: post` — during Setup Flow when one
is created or edited, and in EVERY scheduled run's step for collector-executed channels
(AUTOMATION_SCHEDULING §H). LOAD LEDGER applies. **Read Stage 18 first**: this is the same machine
with a different target and a lower ceiling.

## What this channel is

Stage 18 answers somebody else's post. This one **starts the thread**: a standalone piece of
content in front of the whole group, under the client's name.

That is the most exposed action this system performs. It is what members report as spam, many
groups hold it for an admin, and one careless post costs the account its standing in a group the
operator spent months joining. Everything below is shaped by that.

- **Capacity is per ACCOUNT, not per group** (`posts_per_account_per_day`, default 2). Listing
  more groups buys more comments; it buys **no** extra posts. The exposure of a standalone post
  does not divide by how many groups the operator listed.
- **The same text is never published twice**, in any group. A group post has no permalink to
  dedupe on, so the TEXT is its identity: `draft post` fingerprints the body and refuses a repeat
  with `already_drafted`. Cross-posting one piece into five groups is the single fastest way to
  be recognised as a spammer, and the code refuses it.
- **Approval is the command**, exactly as Stage 18: the bridge publishes on approval, spaced by
  the operator's publish gap (`/ui/settings`, default 5 minutes).
- **A commented or posted-to group never creates CRM contacts.** Same ruling as Stage 18.

## Every scheduled run — what the agent does

### 1. Read the goal

`campaign get --slug X` → `goal.description`. It says what this client should be known FOR. A post
that does not serve it is not written, however good the idea.

### 2. Read the room, group by group

For each url in `audience.groups`:

```json
{ "name": "read <group> before posting",
  "url": "<group_url>", "platform": "facebook", "capability": "fb.group.posts",
  "inputs": { "group_url": "<group_url>", "within_days": 7, "max_pages": 3 } }
```

A wider window than Stage 18 on purpose: a comment needs a fresh post to answer, a post needs to
know **what this group has been talking about for a week**. Read for two things:

- **What recurs.** The question asked three different ways in a week is the post worth writing.
- **What earns engagement HERE.** Compare like/comment counts within this group only; a number
  that is high in one group is invisible in another. Note the shape that works — a question, a
  short lesson, a walkthrough — and write in it.

Never post into a group that is not in `audience.groups`; `draft post` refuses it.

### 3. Decide whether there is anything worth posting

Answer in order, stop at the first "no":

1. Does the goal cover it?
2. Is it useful **without** the client's service existing? If the post only makes sense as an
   advertisement, do not write it.
3. Has this group already seen this from us? Check `commented_posts.json` and the recent sent
   drafts. Repetition is what turns a known name into a blocked one.
4. Would a member who never hires anyone still be glad it was posted?

**Writing nothing is a valid, frequent outcome.** With a ceiling of two posts a day, most runs of
most campaigns should draft zero or one.

### 4. Write it

- Open with the thing the reader gets, not with who you are.
- One idea per post. A list of five tips is a post nobody finishes.
- No link, no price, no "DM me", no call to action beyond the idea itself. The profile is the
  call to action.
- Write it so it stands on its own if a stranger reads it out of context — because they will.

### 5. Deposit it in the approval queue

```
tool crm-store --client-dir {outreach} draft post --campaign X --json '{
  "group_url": "<group url from audience.groups>",
  "body_text": "<the post>",
  "basis": "<one line: what in the group's last week made this the post to write>" }'
```

`tool crm-store draft capacity --campaign X` says how many more will be accepted. Refusals are
answers, not obstacles: `group_not_in_campaign`, `already_drafted` (that text has been written
before), `capacity_horizon_full`, `campaign_paused`, `channel_mismatch`, `no_capacity`.

### 6. Publishing is NOT the agent's step

The operator approves; the bridge publishes with `fb.group.post` and enforces
`posts_per_account_per_day` and the publish gap. The agent **never** enqueues `fb.group.post`.

**Two outcomes, because there are only two we control:**

| Outcome | What it means | What the system does |
|---|---|---|
| posted | the post left our hands — live, or submitted to a group that holds new posts for an admin | draft → `sent`, text remembered forever |
| `error` | nothing was posted (`not_a_group_url`, `group_mismatch`, `ambiguous_composer`, a block) | draft → `blocked` with the collector's own reason, and the text released so it may be rewritten |

Whether a group's admin then approves, holds or rejects the post is **not our business and not a
state this system tracks** (operator ruling 2026-08-17). Tracking it would create a queue the
operator can neither see nor act on. Never re-post something that came back submitted: posting it
again is how one idea becomes two identical posts the moment an admin releases the first.

## The run reply — one line per post campaign

> `{campaign} — read {N} posts across {G} group(s), {D} drafted ({R} refused: reason), {A}
> awaiting your approval, capacity {C}/day.`

Report the refusal reasons out loud. A run that drafts nothing because the ceiling is full is a
healthy run and must say so.

## Storage (Stage 7 addendum)

Same files as Stage 18. The draft carries `channel: "post"`, `group_url`, empty `post_url`,
`to` = the group url, and `hooks_used: [{type: "group_reading", evidence_url: <group>}]`. The
dedupe index keys a post as `body:<fingerprint>` instead of a permalink.

## Worked shape (fictional — never copy the specifics)

A campaign for a mobile bicycle-repair service reads its group for a week and sees the same thing
three times: people asking why a tune-up quote varies so much. The post explains what a shop is
actually charging for and when the cheap option is the right one — useful whether or not anyone
ever calls this business. The second idea that week is a photo tour of the van; it is skipped,
because it only makes sense as an advertisement.

The industry above is fictional on purpose. If a real client's industry appears in this file's
examples, it will be copied into live output — that has happened four times.
