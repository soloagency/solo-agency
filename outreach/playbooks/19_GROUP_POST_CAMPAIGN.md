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

### 1. Load what you are allowed to say

Four inputs, and a post written from fewer than all four reads like anyone could have written it:

1. **The client profile** (`clients/{slug}/.../client_profile_*.md`) — who this client is, what
   they actually do, and what they have actually done. This is the material you may draw on;
   nothing outside it may be asserted. Without this step the rule "no claim the client's own
   experience cannot support" tells the agent what it must NOT say while never telling it what it
   MAY — so it improvises, and improvisation reads as generic.
2. **`goal.description`** (`campaign get --slug X`) — what this client should be known FOR. A post
   that does not serve it is not written, however good the idea.
3. **The group's name** — see step 2.
4. **The client's own recent posts for this campaign**, so today's is not last week's rephrased.

`goal.message_bank` and `goal.cta` are deliberately NOT read on this channel (operator ruling
2026-08-17) — they belong to email, and the campaign page hides them here.

### 2. Know the room — identity, not a week of reading

A group post does not answer anybody's post, so mining a week of the feed for "what recurs" is
work this channel does not need (operator ruling 2026-08-19). What it DOES need is two facts
about the room, and both come from one cheap read the first time you meet a group:

```json
{ "name": "identify <group>",
  "url": "<group_url>", "platform": "facebook", "capability": "fb.group.posts",
  "inputs": { "group_url": "<group_url>", "max_pages": 1 },
  "allowed_extension_instance_ids": ["<instance_id>"] }
```

- **The group's NAME**, from `items[].group.name` — every post record carries it. It is the most
  specific thing you will ever learn about a group and the url tells you none of it: a bare
  `/groups/764877593708803` is a number, while "Help for Insurance Agents" tells you the field,
  that the readers are AGENTS rather than customers, and therefore the register — a peer talking
  to peers, not a vendor talking to buyers. Writing without it is writing blind. Pass it as
  `group_name` when you deposit the draft.
- **Which account can act there**, because only the account that read the group is proven to be a
  member (a non-member is served the page with no composer at all). Pass that same instance id as
  `collector`.

Once a group's name and account are known, **do not read it again** to write another post. Re-read
only when you do not have the name, or when a publish failed in a way that suggests the account
lost access.

Never post into a group that is not in `audience.groups`; `draft post` refuses it.

### 3. Decide whether there is anything worth posting

Answer in order, stop at the first "no":

1. Does the goal cover it?
2. Is it useful **without** the client's service existing? If the post only makes sense as an
   advertisement, do not write it.
3. Has this group already seen this from us? Check the recent sent drafts for this campaign —
   the code refuses the same TEXT twice, but a fourth post rephrasing the same idea is what turns
   a known name into a blocked one, and no code catches that.
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
  "group_name": "<the group's name, e.g. from items[].group.name>",
  "basis": "<one line: why this is the post to write for THIS room>",
  "collector": "<the extension instance id that read this group>" }'
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
