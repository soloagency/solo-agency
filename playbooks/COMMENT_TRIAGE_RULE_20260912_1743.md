# Comment Triage Rule

What: batch classifier for Decision 3's harvest job — decides keep or drop for each comment-thread author, run by the lowest model (Haiku on Claude, the smallest Codex model) in batches of 40 after a code prefilter, only when the Boss orders a harvest run, never in the daily run.

Who loads it: the harvest job (post-approval on a `discovered` source in the client's source registry, `playbooks/LEAD_QUALIFICATION_RULE.md` Step 5); the regression test (`playbooks/tests/lead-rule/`, `score_comments.py`).

Edit policy: any change to this file requires re-running `playbooks/tests/lead-rule`'s comment-triage test (`score_comments.py` against `baseline/comment_triage/`) per its `RUN.md` and attaching the resulting output before the change is considered done — do not edit this rule and move on without it.

The text below is copied verbatim from the owner-approved source (`comment_triage_v7.md`, measured with Haiku on 24 threads × ~40 authors: 90–94% exact keep/drop, keep-recall 91–92%, keep-precision 96–97%, competitor drop 98%, noise drop 99%; batch 40 is the sweet spot — batch 25 loses rows, batch 60 loses 3 points). Do not paraphrase it.

---

# Comment Triage Rule (v7) — keep or drop the authors of a comment thread, in batches

You receive ONE client buyer profile, ONE source post, and a BATCH of comment rows from that post (each row = one author, with every comment they wrote in this thread, plus their author line if the platform shows one). Decide for each row whether this AUTHOR belongs in the client's CRM. Judge the author, not the comment's politeness or topic. Answer every row by id; never skip one.

## Inputs

- CLIENT buyer profile: `sells`, `sells_to`, `types` (memberships, category first, examples after "e.g."), `why_they_need`, `location`, `competitors`, `not_buyers`. An activity the product helps with is never a condition of membership.
- SOURCE POST: platform, community, the post text, and `comment_source_reason` — why this thread was flagged as a place where the client's buyers gather.
- ROWS: `id`, `author_line` (subtitle, bio or headline if shown, else empty), `comments` (all of this author's comments in the thread, in order).

## Step 1 — WHO is this author? (fit)

**1a. `stated_trade`.** Copy, verbatim, any trade, business or role the author states about THEMSELVES — in the author line ("Realtor® | eHomes", "Loan Officer NMLS#…", "Financial Advisor") or in a comment ("tiệm em", "my clients", "I run a gym", "bên em chuyên…"). Write it in `stated_trade`, or "none". Do this BEFORE reading what the comment is about.

**1b. Membership from the stated trade.** If `stated_trade` fits a category in `types` (the category word before "e.g." is the test — "ANY trade", "ANY small business" — the examples are never a closed list), then `sells_to_match` is that line and `fit = high`. The topic of the comment cannot lower it: a realtor relaying an insurance contact, a lawyer congratulating the poster, an advisor giving off-topic advice, an agent saying "not licensed there, following" are all members by who they are. "Not a buyer of X", "not selling X", "not about X" are never reasons. A brand or employer next to a commission or licensed trade ("Realtor at Compass", "State Farm agent", "Insurance Agent, Austin TX") is still an independent professional; only a salaried role at a large company with no pipeline of their own is an "employee". Read `location` literally: "worldwide" / "never disqualifies" means an out-of-area author stays high; a stated area lowers fit only when the row actually places the author outside it — never guess.

**1c. Membership from the thread, when there is no stated trade.** The thread tells you who is speaking:
- Under a request for a provider, an author who offers to do the job ("I can help, DM me", "sent you a message", "call me", a quote, an address) is presenting themselves as that provider → that trade.
- When the post ADDRESSES one audience ("Coaches —", "Chủ tiệm nail nào…", "SaaS founders —", "agents:") or sits in a group of one trade, an author who answers as a peer — sharing their own practice ("my bottleneck", "my clients", "tiệm em", "I batch film weekly", "we keep telling our clients…") — is a member of that audience → that trade or business, even without an author line. A small team ("3 of us") is still a small business.
- An author who states a situation named in `types` is a member by situation: a buyer with a committed plan ("saving for a down payment, looking next year"), a mover with a date, a person with a life change. Undecided ("weighing rent vs buy", "not sure yet"), "signed another lease, waiting a year", and a settled transaction with nothing ahead ("bought two years ago", "sold last spring", "refinanced last month, relieved") are `medium` — unless `types` says a settled case still counts.
- A bare need in the thread's own topic ("me too, I need one", "following, cũng đang cần") shows the SITUATION the post is about; it is `high` when that situation is a `types` membership for this client, otherwise `medium`.
- Judge only what the row states; when nothing shows a trade, a practice or a situation, use the lower tier.

- `fit = high`: a real match in `types` by 1b or 1c, with `location` as worded.
- `fit = medium`: adjacent to every category in `types`; a matching type whose location is truly unclear when location matters; undecided or settled situations; a peer answer too thin to show a practice or situation.
- `fit = low`: matches none of `types` — but check `types` before calling anyone "a consumer": a person described by `types` is never "just a consumer". Low also covers a relay speaking only for someone else with no trade or situation of their own, a tagger, the poster's friend or family, a salaried employee, or no evidence at all.

Relay vs provider: a person recommending someone else is not that someone; the recommended person is not in this thread. A relay with a `stated_trade` is judged on that trade (1b).

## Step 2 — competitor or noise? (before the decision)

- `competitor` — a strict test in two parts, both required: (i) name what the author's own business sells (`author_product`) and name what the client sells (`sells`); (ii) they are the SAME KIND of product or service, sold to the same people, or the author's product appears in `competitors`. Insurance is not marketing; lending is not video editing; "offers a service" is never enough. A videographer IS a competitor for a video-editing app when `competitors` lists video editors; an insurance agent is NOT a competitor for a marketing service or a video app. Under a request for a realtor, every realtor answering is a competitor for a realtor client — and a lead for a client that sells TO realtors. Decide from this client's profile only.
- `noise`: a bot or spam link, a meme, an argument, a bare "following"/"bump"/"👀", a tag with nothing else, a job seeker, a student, the poster wrapping up their own thread (broad thanks, "will try that", no new fact about themselves), or a comment that negates any interest. A reaction word plus a real fact (a need, a place, a trade) is judged on the fact, not the reaction word.

## Step 3 — decision

- `keep` = `fit` is `high` AND not `competitor` AND not `noise`.
- `drop` = everything else. A `medium` fit is `drop` in this pass; say so in `reason` so a human can override.

Intent is NOT required: a right-type author with no stated need is exactly who this pass collects (they enter the CRM as `warm`). If a kept author shows a need for `sells`, set `intent` to `explicit` or `implied`, else `none`.

## Examples (examples, not a list)

1. Thread "need a realtor to list my house in Westminster". Row: author line "Realtor® | eHomes", comment "I know a good insurance agent, will tag you". stated_trade = "Realtor". For a client whose `types` include independent professionals of any trade (a marketing service, a video app, a small-business insurer in that state): fit high; author_product = real-estate brokerage ≠ the client's product → not a competitor → keep. For a realtor client: same product → competitor → drop.
2. Thread "which insurance agent in San Jose?". Row: "Insurance Agent | Allstate", "DM me, I can quote today". For a marketing service selling to independent professionals: fit high; insurance ≠ marketing → keep. For an insurance client: competitor → drop.
3. Thread "Coaches — how are you getting clients from Instagram?". Row: no author line, "I batch film once a week, editing is my bottleneck". Peer answer to a post addressed to coaches → a coach → for a video app or a marketing service: fit high → keep.
4. Thread "who else is buying their first home in OC this year?". Row: "Still renting, saving for a down payment, looking seriously next year" → committed plan → for a realtor client: keep. Row: "Bought two years ago in Costa Mesa" → settled → medium → drop. Row: "Moving to Irvine in August, not sure if we rent or buy" → undecided → medium → drop.

## Output (JSON array, one object per row, same order as the input)

`{"id": "...", "stated_trade": "verbatim or none", "person_type": "…", "sells_to_match": "… or none", "author_product": "what their business sells, or none", "fit": "high|medium|low", "competitor": true|false, "noise": true|false, "intent": "explicit|implied|none", "decision": "keep|drop", "reason": "one line"}`
