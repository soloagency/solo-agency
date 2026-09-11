# Lead Qualification Rule (v11) — Fit × Intent

What: the single rule of record for deciding whether any item — a post, comment, caption, or people-row — is a lead, and what kind. It replaces separate ad hoc lead heuristics with one general test, industry-independent, run against each client's own `buyer_profile`.

Who loads it: Stage 10 Detection Workflow (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`); the Social Discovery Pass (feed/search, people-search rows on Facebook, Instagram, X); the lead-engine skill's extractor tier (`playbooks/skills/lead-engine/`); Step 5 below, which records a post as a `discovered` source in the client's source registry (`tool source-registry`, kind `discovered`) for the Boss to approve — nothing is harvested automatically; the harvest job that runs only on the Boss's explicit order (`playbooks/COMMENT_TRIAGE_RULE.md`, a separate rule applied to the thread's individual authors once a discovered source is approved); and the regression test (`playbooks/tests/lead-rule/`).

Edit policy: any change to this file requires re-running `playbooks/tests/lead-rule` per its `RUN.md` and attaching the resulting report before the change is considered done — do not edit this rule and move on without that report.

The text below is copied verbatim from the owner-approved source (`rule_v11.md`, tested blind on 120 scenarios × 5 clients: 96.2% lead/non-lead accuracy, 93.8% exact tier, 96% of profession-only posts kept as warm, 100% competitors caught). Do not paraphrase it.

---

Qualify ONE item (a post, comment, caption, or profile row) for ONE client. Answer three questions in order — WHO is this person, competitor or noise, WHY NOW — then read the decision off the matrix. Reason from the item's own language, never magic phrases; never require a stated need to decide who someone is.

## Inputs

- CLIENT buyer profile: `sells`, `sells_to` (who buys, as membership types and situations), `types` (the same memberships, one per line — pick the closest), `why_they_need`, `location` (if it matters), `competitors`, `not_buyers`. An activity the product helps with (making video, buying insurance, marketing) is never a condition of membership.
- ITEM: platform, surface, the community it was found in, the author line (subtitle, bio or headline, if any), the text, a date note.

## Step 1 — WHO is this person? (fit)

First name the person: `person_type` = what they are, in a few words (trade, role, business or situation), inferred from every clue — what they do, own or run, where they are, their stage of life, the community, handle, hashtags, language. Infer the ordinary case: a listing post is a real-estate agent; quoting rates is a loan officer or insurance agent (self-employed, own pipeline, even if lender-affiliated); salon work is a salon owner; app revenue is a founder. A call to action aimed at THEIR OWN customers ("DM me for a quote") is evidence of what they do, never their own need.

Then run the MEMBERSHIP TEST: go down THIS client's own `types` (never another client's) and pick the closest line as `sells_to_match` (or "none"), asking only "is this person one of those?" — answered from who they are. The CATEGORY before the colon or before "e.g." is the test; the names after are examples, never a closed list — "any trade" also matches a dentist, an escrow officer, a videographer. "No evidence of doing X" (making video, marketing, buying coverage) is forbidden as a reason. Selling a service to others (coaching, an agency, freelancing) does not itself lower fit or make them a competitor — Step 2 decides that.

- `fit = high`: `sells_to_match` is a real match, and, when `location` matters, the person is inside the service area or arriving into it (a stated move there, a job there, a date) — arriving counts as inside for LOCATION.
- `fit = medium`: an adjacent profession or situation; the right kind of person whose location is unclear when location matters (an exact match with no location stays medium); a situation already settled with nothing ahead for a service tied to one transaction ("just bought") — except where `types` says a settled case still counts; or a real change with no named transaction where `types` requires one (arriving alone isn't a named purchase or lease).
- `fit = low`: no match in `types`, evidence of being outside `sells_to`; a plan hedged as possibly not happening ("maybe X or maybe not") counts here, not as medium; or a different specific situation than the one described (refinancing is not a home purchase).

Community context: the author's OWN words decide their profession — strip the group name and ask whether the TEXT ALONE shows what THEY DO for others, versus only something that happened to THEM personally; a personal post ("I'm stuck with") inside a professional-sounding group does not by itself make the author that profession — treat as unknown/consumer unless the text shows the trade. For LOCATION, a community whose name states a place ("Orange County", "California") is evidence of place; a trade or national name ("Realtors USA", "Việt") is not. A bio-only profile row is judged on its own claim; a missing post never lowers fit. Uncertainty about timing lowers fit to medium; uncertainty about employment structure or exact trade within an "ANY" line does not.

## Step 2 — Competitor or noise? (decide this BEFORE intent)

1. `competitor`: sells the same thing as `sells`, to the same people and market — another agent, broker, officer, agency, app, tool or provider (same trade outside `location`/`not_buyers` is not a competitor). Ask: "whose problem does their work solve, their own or a paying customer's?" A freelancer or agency selling THIS service to others is a competitor; doing the same work only for their own business is not. This runs before `not_buyers` and before the matrix: a same-market seller is `competitor` even with a shown need — unless `sells_to`/`types` names that profession as a customer or partner, then treat as a customer.
2. `none` (noise): the person cannot buy this offer in any reading — a job seeker, a student, spam or multi-level selling, a meme or political post, the client themselves; or the text negates the need in any language ("not looking", "already have this", "đừng pitch nữa") — not merely solving a related problem another way; or the item is stale when timing matters. A high-fit person who explicitly rejects this kind of offer is `none`, not `warm`.
3. Referral partner: sends buyers rather than buying (an agent for a lender, a lender for an insurer) — a lead only when `types` names that role; `none` when `not_buyers` names it; `watch` when neither does. Check only this client's own roster, never another's.

## Step 3 — WHY NOW? (intent)

Intent is THIS PERSON's own need for `sells`, now or soon. Before writing `explicit` or `implied`, NAME the friction in `why_they_need` that this content shows; if you cannot name one, intent is `none`. The situation must call for `sells` itself, not a neighbouring purchase. An ask for a DIFFERENT provider in an unrelated domain (a lender when this client insures) blocks EXPLICIT intent only, not an implied trigger named elsewhere in the text, and never applies when the ask sits inside this client's own offer (video editing, for a video/content client). A CTA to the person's own customers is never intent, even as active soliciting. A bio-only row has `intent = none` unless the bio itself states a change or a need. For a partner segment, intent means they look for, compare or complain about a partner of this kind; their normal work (a listing, a closing) is `none`.

- `intent = explicit`: they ask for, look for, compare, evaluate or complain about exactly this kind of offer or provider, for themselves or their business.
- `intent = implied`: a gap or change that makes THIS need likely soon — a new thing to protect, a new business with no customers yet when `why_they_need` is about finding customers, "no time to edit", "premium doubled" — in a post or a bio line alike. Decision question: name the new thing created (asset, dependent, employee, address, policy, loan, deadline, problem) AND the friction it produces; more of the same work at greater volume, nothing new, stays `none`. When the new thing IS exactly what `types` lists as this client's own trigger, naming it already names the friction — no separate complaint is required.
- `intent = none`: nothing points to a need for `sells` now. Routine work is `none`: a listing, a rate update, a client win, an open house, a bio that only says what they do. Test: "Would this be true on any ordinary working day for this person?" If yes, `none`. Growth of an ALREADY-RUNNING business — a new product line, added hours, a milestone — stays `none` even naming general tiredness: the struggle must name THIS client's own friction (posting, finding customers, coverage), not day-to-day overwhelm. But the FIRST launch of a whole new business with no customers yet is `implied` wherever `why_they_need` is about finding customers, however proudly phrased. Running one's business is their job, not a gap; still deciding with no committed plan is `none`.

A person can be a perfect customer and have `intent = none` today. That is normal and still valuable: they become `warm`.

## Step 4 — Decision (the matrix, the same for every industry)

| fit | intent explicit | intent implied | intent none |
|---|---|---|---|
| high | **hot** | **hot** | **warm** |
| medium | **warm** | **watch** | none |
| low | **watch** | none | none |

Compute fit and intent independently, then look up that cell — `watch` is not a catch-all for "uncertain". `competitor`/`none` from Step 2 override it.

## Examples (examples, not a list)

1. Agent posts "Just listed, open house Saturday, DM me." For a marketing service, a video app, or a lender that counts agents as partners: fit high, intent none (routine) → warm. For another agent nearby: competitor. If she instead posts "one more marketing agency DMs me, I'm blocking them" — explicit rejection → none.
2. Salon owner writes "hired my first employee, so much paperwork." For a small-business insurer: fit high, intent implied (a new employee changes what is insured) → hot. For a marketing service: fit high, intent none (no customer-finding friction named) → warm.
3. Bio only: "Licensed insurance agent, bilingual, Garden Grove". For a video app: fit high, intent none → warm. For a lender: no match → none.
4. Homeowner asks "any good loan officer for a refinance?" For a lender: fit high, explicit → hot. For an insurer: no independent trigger named → none. For a realtor: fit low (staying put) → none.
5. "Accepted a job in Irvine, moving from Chicago next month." For a realtor serving Orange County: fit high; implied → hot. For a lender: fit medium (buy or rent not named) → implied → watch.

## Step 5 — Is this post a lead source in its comments? (comment_source)

Independently of Steps 1–4, decide whether the people who ANSWER this post will be the client's buyers. Walk these questions in order, record each answer in `comment_source_reason`, and stop at the first question that settles it.

1. **Does the post ASK for replies from others?** Yes: a request for a provider, a recommendation, a referral, a quote, an expert answer; a question only one trade can answer; a peer question addressed to one audience ("Coaches — how do you…", "Fellow agents…"); a gathering question ("who else is moving to Irvine this summer?", "first-time buyers, what surprised you?"). No: a statement or a PROMOTION — a listing, a showcase, an update, a rate note, a launch, a milestone, market commentary, a complaint with no question, a recruiting ad, and any call to action that offers the author's OWN service ("anyone need an editor, DM me", "doors open in January", "ai cần pre-approval nhắn mình", "tag someone who is looking"). The author of a promotion is selling, not asking; the people who answer a promotion are that author's customers, not a gathering the client can harvest. Statement or promotion → `unlikely`, stop.
2. **Name the answerer type.** Who will reply — a trade ("realtors", "loan officers", "personal trainers", "tutors", "plumbers") or a situation ("first-time buyers in Orange County", "new parents in Irvine", "Tesla owners in California", "movers into Irvine")? Use the post's own words. A request for a provider is answered by providers of that trade; a peer or gathering question is answered by that audience; a question about a purchase, a policy, a loan is answered by peers in that situation AND by providers of that trade — name both.
3. **Membership test — write `types_match`.** Go down the client's `types` and copy the line that the answerer type belongs to, verbatim (or "none"). Rules for matching:
   - Answerers who are PROVIDERS of any trade (people who sell a service or run a business, whatever the trade) match any line whose category word is "independent professional of ANY trade", "self-employed professional of ANY trade", "owner of ANY small business", "creator, course seller, freelancer of ANY kind" — regardless of the client's own product. A loan officer, a personal trainer, a tutor, a plumber, a videographer answering a request are self-employed professionals; they match such a line for a marketing service, a video app AND a small-business insurer alike.
   - Answerers in a SITUATION match a line that names that situation (buyers, movers, new parents, a new asset, a rising premium, a new address, holders of a mortgage).
   - "none" → `unlikely`, stop.
4. **Own-trade check (runs even after a match).** If the answerer type sells what the client sells — realtors for a realtor client, lenders for a lender client, insurance agents for an insurance client, video editors for a video-editing app — they are competitors → `unlikely`, unless `types` names that trade as a partner or referral segment, then `likely`. When question 2 named both peers and providers: a post that ASKS FOR a provider ("ai biết agent bảo hiểm nào…", "recommend a lender") is answered by that provider — apply this check to the provider; a peer or gathering question with no provider ask ("có ai bị vậy không?", "who else is refinancing?") is answered by the peers — apply it to the peers.
5. **Location.** When `location` matters and the post places the answerers outside the client's area with no move into it → `unlikely`. A post whose place is unknown or inside the area → `likely`.

`comment_source` is a property of the THREAD, not of the author: a consumer's post can be `none` for its author and `likely` as a source. Nothing is harvested automatically from a `likely` post; it is only recorded for the Boss to approve.

Examples (examples, not a list): "Cần tìm realtor để listing nhà ở Westminster" — Q1 request; Q2 realtors; Q3 `types_match` = the "ANY trade" line for a marketing service, a video app, a small-business insurer in that state, and the partner line for a lender; Q4 for a realtor client → competitors → unlikely. "Anyone have a personal trainer in Irvine?" — Q2 trainers → the same ANY-trade clients → likely; a realtor or a lender → no match → unlikely. "Premium bảo hiểm xe tăng gấp đôi, có ai bị vậy không?" (no provider asked) — Q2 peers with a rising premium; for an insurer: they match "rising premium" → likely; for a marketing service: peers are consumers → no match → unlikely. "Ai biết agent bảo hiểm nhà tốt giới thiệu dùm?" (a provider asked) — Q2 insurance agents; for an insurer: own trade → unlikely; for a marketing service or a video app: agents match ANY trade → likely. "Who else is moving to Irvine this summer?" — Q2 movers → likely for a realtor, a lender and an insurer that names a new address; unlikely for clients that sell to professionals. "Just listed 4bd in Irvine, open house Saturday" and "Anyone need a fast editor? DM me" — Q1 promotion → unlikely for everyone.

Output adds three fields: `"types_match": "verbatim line or none", "comment_source": "likely|unlikely", "comment_source_reason": "Q1 … Q2 … Q3 … Q4 … Q5 …"`.

## Output (JSON, one object per item)

`{"id": "...", "person_type": "what this person is, in a few words", "sells_to_match": "the closest line of types, or none", "fit": "high|medium|low", "fit_reason": "one line", "intent": "explicit|implied|none", "intent_reason": "one line", "decision": "hot|warm|watch|none|competitor", "types_match": "verbatim line of types matching the comment-section answerer, or none", "comment_source": "likely|unlikely", "comment_source_reason": "Q1 … Q2 … Q3 … Q4 … Q5 …"}`
