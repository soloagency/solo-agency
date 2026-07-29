# Brief → Email — the derivation chain

Load rule: load AFTER running `tool crm-store draft brief` for the lead, alongside `weave.md`.
This module is the WRITING PROCEDURE; `weave.md` is the rhetoric it drives. LOAD LEDGER applies.

## Who writes — the writer runs ALONE, in a fresh context

The orchestrating session (the daily run, with its ledgers, queues and tool loops) **never writes
body text inline**. Writing is a SEPARATE, fresh-context call — one writer per lead, or one per
batch of at most 5 — whose entire input is exactly two things: this module and the lead's brief
(plus `weave.md`). Nothing else. No ledgers, no queue state, no other leads' drafts.

This is measured, not stylistic preference. The same skill and the same briefs produced the
operator-approved batch when written by fresh-context writers (13/13 pass every style gate,
longest summary overlap 7 words, zero repeated sentences) and produced 22/22 "anh/chị", 18
sentences repeated 208 times, and a pasted English summary when written inline from a long
orchestration loop. By the thirtieth email of a loop, the rules sit a hundred thousand tokens
back and the cheapest continuation is a template. No prose survives that; isolation does.

The orchestrator's job around each writer call is mechanical: run `draft brief`, hand the writer
the brief + this module, take back `{subject, body_text, pain_addressed, hooks_used}`, run
`draft write`, and on a refusal send the FULL error message back to the same writer for a rewrite
— never patch the body itself, and never retry by pasting.

The style gates (`vn_register`, `name_overuse`, `template_sentence`, `summary_pasted`,
`phrase_stuffing`) are TRIPWIRES, not a workflow: with an isolated writer they are expected to
fire ZERO times. A firing means the writing setup is wrong (stale summary language, inline
writing, degraded context) — fix the setup, don't iterate against the gate.

## The contract

The brief is the only source of facts. It carries: the client profile sections
(`business_description`, `offer`, `icp`, `pain_points`, `value_prop`, `proof_points`,
`brand_voice`, `language`, `compliance_notes`), the campaign goal (the operator's own
`description` plus the derived fields), the key-message rotation state for THIS lead, the lead's
hooks + `writing_brief`, and every earlier touch. If a fact is not in the brief, it does not go
in the email. If something in the brief contradicts a derived field, the operator's `description`
and the profile win; flag the mismatch in your run report.

The operator was asked ONE question (the goal, in their words). Everything else you need was
collected at setup and is IN the brief. That is the deal: they answer little, so you must use
everything. An email that ignores the profile while the brief carried it is a broken promise,
and the draft record shows it (`pain_addressed` empty, `Teaches` line missing).

## The chain — seven moves, in order, before you write a word

1. **Read the person.** From `icp.segments` + the hook, write ONE internal sentence: who is
   this, and what are they visibly trying to do? Pick the segment they match; you will declare
   it via `pain_addressed`'s framing. (A Vietnamese-language profile serving Vietnamese clients
   → the Vietnamese-American segment → the email is written in Vietnamese. `language` in the
   brief decides register; never literal-translate English marketing phrasing.)

2. **Pick ONE pain.** Intersect the profile's `pain_points` (they carry per-segment relevance)
   with what the hook shows. The hook is evidence of what they already DO; the pain is what
   doing it costs them or what doing more of it would take. One pain, named concretely in their
   terms. Declare it in `pain_addressed`. If no pain intersects, take the segment's
   highest-relevance pain and frame it as an observation about the field, never an accusation
   about them.

3. **Open with the hook USED.** Say what their content SAYS, in your voice, with its specifics
   (the street, the topic, the number, the claim they made). Never paste the dossier summary
   (it is a third-person internal note). Then apply the swap test before moving on: if another
   lead's hook could replace yours and the next sentences still stand, you have complimented
   the hook, not used it. Rewrite until the swap breaks the email.

4. **Reframe through a key message.** One FRESH message from `bank_rotation.fresh_for_this_lead`
   reads the hook: "here is what your content shows, that most people in your field miss." This
   is the teach — the only part of the email a competitor could not also send. The gate refuses
   an email that lands no bank message (`no_key_message`) and one that re-teaches a message this
   lead already got while fresh ones remain (`rotate_bank`).

5. **The mechanism.** One concrete sentence on WHY the offer resolves THAT pain for THAT
   segment. Causal, not feature-list: "a team that scripts and edits for you" resolves "no time
   to publish consistently"; it does not resolve "camera anxiety" (preparation and coaching do).
   Wrong pairings read as generic instantly. Source: `offer.items` × your chosen pain.

6. **Proof, or soften.** A factual claim takes a `proof_point` (claim + evidence_url) from the
   brief. No proof point → soften to experience: "what we keep seeing with <profession>s" /
   "điều tôi liên tục thấy". Never invent numbers, clients, or results; `banned_claims` applies.

7. **One small ask — the campaign's ask, not yours.** The CTA is `goal.cta.text`'s intent,
   rendered naturally in the email's language: if the campaign asks them to VIEW the plan, the
   ask is opening the link, and you do NOT append a reply request as a second ask (two asks
   halve each other). `get_reply` ends on the question and carries no link; other types per
   `structures.md`. One ask. Never two. **Then sign — from the DATA, never from an example**:
   the sign-off is the brief's `signature.signature_block`, rendered one `|`-separated segment
   per line (the given name may take its Vietnamese form in any language); if the profile has no
   `signature_block`, fall back to given name + client name. Nothing after the sign-off.
   `draft write` refuses a body missing the given name or dropping the block's website
   (`missing_signature`). A live batch lost its website line because an over-specific example
   HERE beat the profile's own data; the profile decides what the signature contains, this
   module only says where it goes.

Then write the email as ONE argument — the moves are derivation steps, not paragraph headers.
A reader must never be able to see the seams. **Never narrate a move in the email itself**: "the
pain I picked for you is..." / "nút thắt em chọn cho anh là..." / "with you, this lesson shows
clearly..." is the scaffolding read aloud, and one live draft shipped exactly that. The chain
tells you what to derive; the email only ever contains the RESULT.

## Addressing — how the name is used

- **Read the brief first**: `lead.name_greetable` says whether the stored name may follow
  "Chào"/"Hi" at all, and `lead.name_missing` says there is no name. Not greetable (a page name, a
  "Person - Tagline" composite) or missing → neutral greeting ("Chào bạn", "Hi team"). **Never
  invent a name from the profile URL slug** — the enrich skill forbids it and a slug is not a name.
- **A person**: greet by GIVEN name only (`lead.name.given` when the brief carries it) — "anh Liêm", "chị Hà", "Hi Grant" — never the full
  name glued after a gender word ("anh Quang Ngô" is a form letter; "anh Quang" is a person).
- **A page or company**: the name alone, NO gender word ever — "Chào BGV Invest", "Hi team".
  A page is not a person: "anh Sống khoẻ cùng Liêm" shipped in a live draft and reads absurd.
  When the page name embeds a person ("Sống khoẻ cùng Liêm"), address THAT person: "anh Liêm".
- **The name appears at most twice in the whole email** — the greeting plus at most one natural
  mention. The reader knows who they are; a name repeated in every sentence is mail merge, and
  `draft write` refuses it (`name_overuse`, ≥3 occurrences of the stored full name).

## The degradation matrix — what changes when the data thins

The chain holds under every input state; only the moves compress. Never fall back to a template:
when unsure, write SHORTER, not more generic.

| State | What changes |
|---|---|
| Hook rich (curated reel/post with real specifics) | Full chain. The hook's own detail carries move 3; spend your length on moves 4-5. |
| Hook thin (one dated signal, few specifics) | Compress 3 to one clause; the teach (4) becomes the email's center. |
| No hook, fallback allowed (`generic_honest_opener`) | Skip 3. Open with the segment observation (move 2 framed as the field's reality), grounded in a roster/licence fact if one exists. The teach still runs — it is what makes an opener honest instead of empty. |
| Segment unmatched | Use the profession you can SEE (from the hook/profile header), the closest pain, and drop segment-specific claims entirely. |
| No proof point | Move 6 softens to experience. Do not smuggle the claim back in as fact. |
| Vietnamese recipient | Whole email in Vietnamese. Addressing per the rules below; gender READ FROM THE NAME ("anh" male, "chị" female, "bạn" unsettled). NEVER "anh/chị". The teach must read as a peer's insight, not a translated slogan. |
| Follow-up (step ≥2) | New FRESH message (rotation is enforced), new angle on the same pain or the next-ranked pain, reference the prior touch in ≤1 clause, never "just following up", never re-pitch the same mechanism. Reply drafts: answer first, teach second, ask last. |

## Self-check before `draft write`

- Swap test passes (move 3).
- Exactly one pain, declared in `pain_addressed`, in the recipient's language.
- ≥1 FRESH key message genuinely woven (not quoted like a slogan).
- Every claim traces to the brief; anything unproven is softened.
- No dossier-summary pasting; no sentence you have used for another lead this batch.
- House Style: no em dash, no `Re:`/`Fwd:`, subject specific to THIS email's argument.
- Declarations set: `hooks_used`, `pain_addressed`; the companion URL per the goal.
- Signed: the sign-off renders `signature.signature_block` in full (website line included when
  the profile declares one).

`draft write` enforces what it can: `no_brief`, `hook_not_woven`, `no_key_message`,
`rotate_bank`, `vn_register` ("anh/chị", pages greeted as people, and register mixing — greeting
"anh" then switching to "bạn"), `name_overuse`, `template_sentence` (a sentence already used in
another lead's live draft, INCLUDING offer/CTA lines — render them fresh per email),
`summary_pasted` (≥10 consecutive words of the hook summary lifted verbatim, or ≥6 lifted twice —
weaving reuses the content's short key phrase, never a block; if the stored summary is in a
different language than the email, FIX THE DOSSIER SUMMARY first, then draft), and
`phrase_stuffing` (the same 6-word phrase three times in one email is a template variable, not a
person — name the topic once, then use a natural reference). This checklist covers what only you
can judge. The Approval Report prints
Evidence / Teaches / Pain addressed / Briefed per card — the operator sees exactly which of
these you did.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
