# Lead And Competitor Detection

Stage: `10`

## Load Rule

Load whenever the agent is about to detect, score, report, store, or improve lead and competitor opportunities.

Also load during:

- first agency run;
- scheduled daily run;
- private data source scan analysis;
- public research analysis when lead or competitor signals are possible;
- HTML report generation or repair;
- any human request about leads, competitors, comments, replies, opportunities, outreach, or competitor monitoring.

## Hard Gates For This Stage

- Lead and competitor detection is not a light report appendix. Treat it as a core agency function.
- Detect leads and competitors during the same public/private data collection pass used for ideas and market research. Do not add a separate extra scan unless the human explicitly asks.
- For the first lead and competitor pass for a client/source set, use 10 scrolls per approved private data source when Local Collector is active and safe.
- For recurring daily scheduled runs, use 5 scrolls per approved private data source by default, unless the saved Local Collector configuration is lower or the human explicitly configured a safer lower value.
- Source discovery mode is different: it may scroll deeper under Stage 8 rules. Do not confuse source discovery with daily lead/competitor monitoring.
- For private data sources, use only the Solo Agency Local Collector extension plus Local Collector app. Never use Claude in Chrome, Codex built-in browser, Playwright, Puppeteer, Selenium, or any agent-controlled browser for private data source collection.
- Every detected lead or competitor opportunity shown in the HTML report must include the post/current URL when available, context, classification, why it matters, and a copy-ready suggested comment.
- The HTML report must keep public data source opportunities and private data source opportunities separate. Use `Public Lead & Competitor Opportunities` inside `Public Data Source Intelligence` and `Private Lead & Competitor Opportunities` inside `Private Data Source Intelligence`, or the same meaning in the human/report language.
- Suggested comments are for human review only. Do not auto-comment, auto-DM, or initiate outreach without explicit human approval — the **send/act** side always needs a human (see the approve-then-send gate). Data **collection and analysis**, on the other hand, is consented by the operator's own setup + command: the agent may read, extract, and combine whatever the operator directs it to research — the operator's own business data (industry, goal, content) and the prospects/sources they point the collector at, including contact details (email/phone) — for lead-finding and email personalization. **The one absolute prohibition:** never read, store, or transmit the operator's own credentials or secrets (usernames, passwords, cookies, tokens, session/auth data, API keys). Do not bypass access controls or CAPTCHAs — read what the operator's own session already renders.
- Comments must build personal brand and trust by adding value. They must not directly advertise the user's service, attack competitors, or sound like generic AI text.

## Source Preservation Rule

This file is detailed source material. Do not summarize away definitions, scan-depth rules, reporting requirements, comment style requirements, storage fields, or completion gates.

---

## Definitions

### Lead

A lead is a person, account, post, comment, or thread that shows a direct or indirect need related to the product or service the human provides.

Direct lead signals include:

- asking for a provider, expert, quote, recommendation, estimate, or solution;
- describing an urgent problem the user's offer can solve;
- asking what to do next, who to hire, what to buy, or how much something should cost;
- complaining that a current provider, tool, process, or solution is not working;
- comparing options before making a buying decision.

Indirect lead signals include:

- expressing a pain point, fear, objection, confusion, life event, workflow breakdown, or recurring frustration tied to the user's offer;
- asking adjacent questions that usually happen before the buying moment;
- joining a discussion where the same audience is clearly trying to solve a related problem;
- reacting strongly to a competitor post, case study, offer, or educational explanation.

A lead is a signal for the human to review. It is not permission for the agent to contact the person.

### Competitor

A competitor is any person, account, company, creator, product, community, content asset, or alternative solution that competes for the same target audience, buying intent, attention, or trust.

Competitor types:

- `direct_competitor`: offers the same or very similar product/service.
- `indirect_competitor`: solves the same problem with a different service, product, tool, or method.
- `adjacent_solution`: serves the same pain point before or after the user's offer.
- `attention_competitor`: attracts the same target audience even if it does not sell the same thing.
- `authority_or_kol_competing_for_trust`: owns trust, education, or recommendation power in the same audience.

Competitor posts matter because they often attract the same people the user wants to help. The user should either learn from the signal or appear in the discussion with a useful, natural comment.

## Scan Depth Contract

Lead and competitor extraction should happen inside the normal data collection pass:

```text
research/source scan -> data points -> leads -> competitors -> ideas -> best idea -> draft -> report
```

Do not run a second scan just for lead/competitor detection unless:

- the human explicitly asks for a deeper lead/competitor pass;
- the first scan failed or produced too little usable data;
- a saved schedule or config says a deeper first pass is allowed.

Private data source depth:

- First lead/competitor pass for a client/source set: 10 scrolls per approved source.
- Recurring daily run: 5 scrolls per approved source.
- Use `collector_config.scroll_delay_seconds`, defaulting to about 5 seconds.
- Respect `collector_config.max_scrolls_allowed`, account-safety limits, rate-limit warnings, session-expired states, and platform warnings.
- If Local Collector config exists and has lower safety settings, obey the safer lower value and record why coverage is lower.
- If the human configured a higher number, cap it at Stage 8 safety limits unless the collector explicitly supports a safe higher discovery mode.

Human-facing disclosure when scanning private groups/sources:

```text
I will go through each approved group/source one by one and scroll {N} times per source. For the first lead/competitor pass I use 10 scrolls when safe; for normal daily runs I use 5 scrolls. I read this from the Local Collector configuration when available.
```

## Detection Workflow

For every relevant source item, the agent must ask:

1. Does this show a direct or indirect need connected to the user's offer?
2. Does this reveal a pain point, objection, buying trigger, decision moment, or urgency signal?
3. Does this source/post/account compete for the same audience, trust, intent, or attention?
4. Would a useful comment from the human help build presence without selling?
5. Is there a stable post/current URL or source URL that lets the human inspect the context?
6. Is this safe and appropriate to report without exposing unnecessary personal data?

Required lead fields:

- lead label or safe descriptor;
- source/platform;
- source type: public | private;
- profile URL when visible and safe;
- post/current URL when available;
- captured at;
- emails (optional) — email addresses extracted from the page content the collector captured (visible text + `mailto:` links); present when the captured content contains them; empty/absent otherwise;
- phones (optional) — phone numbers extracted from the same captured page content (visible text + `tel:` links), digits normalized (E.164-style with a leading `+` when a country code is present); present when the captured content contains them; empty/absent otherwise;
- evidence snippet or safe summary;
- lead type: direct_need | indirect_need | pain_signal | buying_trigger | objection | comparison | complaint | adjacent_need;
- lead level: hot | warm | watch;
- related offer;
- related pain point;
- confidence: high | medium | low;
- suggested next action for the human;
- suggested value-first comment;
- outreach/compliance note.

Required competitor fields:

- competitor name/page or safe descriptor;
- competitor type;
- platform/source;
- profile URL when available;
- post/current URL when available;
- captured at;
- audience overlap;
- offer/positioning;
- content theme or hook pattern;
- engagement or comment signal;
- threat/opportunity level: high | medium | low;
- what the user can learn;
- suggested value-first comment;
- monitoring action.

## Opportunity Scoring

Lead score dimensions:

- urgency;
- fit with the user's offer;
- clarity of need;
- ability to help without being spammy;
- source credibility;
- location fit when location matters;
- recency;
- confidence.

Competitor score dimensions:

- audience overlap;
- engagement quality;
- repeated pain points in comments;
- positioning strength;
- freshness;
- content pattern usefulness;
- strategic threat;
- opportunity for the user to add a better, clearer, or more helpful perspective.

## Audience Value-First Opportunity Rule

Lead and competitor intelligence must produce useful audience-facing ideas, not direct praise for the user's product/service.

When a lead or competitor signal becomes an idea, best idea, suggested comment, script, blog, caption, or recommendation, it must state:

- the audience pain point or confusion;
- the viewer value / lesson;
- the source signal;
- the non-promotional angle;
- why it helps the audience;
- the soft business relevance to the user's offer.

Do not convert competitor positioning into `the client should out-position them`, `our product is better`, `{Client Product} wins`, `choose us`, or similar direct promotional framing. If the idea cannot teach the audience something useful without selling, reject it as `promotional_not_value_first`.

Bad competitor-derived idea:

```text
MiniMeo out-positions competitors by selling without selling across multiple platforms.
```

Better competitor-derived idea:

```text
How small brands can test whether their "selling without selling" message is actually clear before pushing it across every platform.
```

Use qualitative labels if numeric scoring would slow the run:

```text
high | medium | low
```

## Comment Drafting Rules

For every lead and competitor opportunity in the HTML report, draft one short comment the human can copy.

The comment must:

- use the same language as the post or thread;
- respond to the actual context, not a generic template;
- provide a useful insight, question, clarification, checklist item, or perspective;
- sound like a real person, not an AI assistant;
- be short enough to paste into a social thread without looking like a mini blog post;
- avoid direct selling, self-promotion, `DM me`, `message me`, `inbox me`, `book a call`, `reach out to start`, `we can help`, or service pitches;
- avoid attacking or undermining a competitor;
- avoid guarantees, regulated claims, legal/medical/financial advice, or unsafe instructions;
- avoid over-polished phrasing, generic motivational language, and obvious AI cadence;
- avoid bullet lists unless the platform/context naturally uses bullets.

The comment may include one or two tiny natural imperfections when appropriate, such as a casual spelling choice, a missing accent, or a small human-sounding typo. This is allowed because a slightly imperfect helpful comment can feel more trustworthy than perfectly polished AI-style text.

Do not force typos. Never make the user look careless, rude, uneducated, unprofessional, or unclear. The comment must remain easy to understand.

Good comment pattern for a lead post:

```text
This usually gets easier if you separate the urgent question from the long-term decision. First check what has to happen this week, then compare options after that. A lot of people accidentally mix those two and overpay or choose too fast.
```

Good comment pattern for a competitor post:

```text
One thing I like about this topic is that the "right" answer depends a lot on timing. The same advice can be great before renewal and pretty bad after a notice already arrives.
```

Bad comments:

```text
We help with this. DM me.
```

```text
As an expert in this industry, I strongly recommend scheduling a consultation today.
```

```text
This competitor forgot to mention that our service is better.
```

## HTML Report Contract

The report must include lane-specific sections named:

```text
Public Lead & Competitor Opportunities
Private Lead & Competitor Opportunities
```

If the report is in another language, translate the titles naturally. English defaults:

```text
Public Lead & Competitor Opportunities
Private Lead & Competitor Opportunities
```

A report-level rollup named `Lead & Competitor Opportunities` is allowed, but it must not replace the lane-specific sections and must not mix public and private evidence without visible `source type` labels.

For each opportunity, include:

- type: lead | competitor | both;
- classification, such as hot lead, warm lead, direct competitor, indirect competitor, attention competitor;
- source/platform;
- post/current URL as a visible link when available;
- profile/account URL when visible and safe;
- safe context summary;
- why this matters;
- suggested human action;
- copy-ready comment;
- a real working copy button for the comment.

Copy buttons are allowed because they perform a real local browser action. They must copy only the suggested comment text. They must not imply that the comment will be posted automatically.

Example HTML behavior:

```html
<button type="button" class="copy-comment" data-copy="Helpful comment text here">Copy comment</button>
<script>
document.querySelectorAll('.copy-comment').forEach(function (button) {
  button.addEventListener('click', async function () {
    var text = button.getAttribute('data-copy') || '';
    try {
      await navigator.clipboard.writeText(text);
      button.textContent = 'Copied';
    } catch (error) {
      var area = document.createElement('textarea');
      area.value = text;
      document.body.appendChild(area);
      area.select();
      document.execCommand('copy');
      document.body.removeChild(area);
      button.textContent = 'Copied';
    }
  });
});
</script>
```

If no opportunities were found, the report must distinguish:

- no leads/competitors found after scanning;
- not scanned because private data sources are pending Local Collector activation;
- not scanned because source/session was unavailable;
- public data sources only run with limited private lead coverage.

## Storage Contract

Store lead and competitor opportunities in both the existing lead/competitor logs and the unified opportunities ledger when possible.

Recommended unified ledger:

```text
history/YYYY-MM/lead_competitor_opportunities.jsonl
```

Recommended JSONL fields:

```json
{
  "date": "YYYY-MM-DD",
  "client_slug": "client",
  "opportunity_type": "lead",
  "classification": "hot_lead",
  "source": "Facebook Group",
  "source_type": "private",
  "platform": "facebook",
  "profile_url": "https://...",
  "post_url": "https://...",
  "emails": ["name@business.com"],
  "phones": ["+14155550100"],
  "captured_at": "ISO-8601",
  "safe_context_summary": "Short summary",
  "evidence_snippet": "Short visible snippet if safe",
  "why_it_matters": "Reason",
  "related_offer": "Offer",
  "related_pain_point": "Pain point",
  "confidence": "medium",
  "suggested_action": "Human reviews and decides whether to comment",
  "suggested_comment": "Value-first comment",
  "comment_language": "en",
  "comment_style_notes": "natural, short, no direct pitch",
  "status": "needs_review"
}
```

The optional `emails` and `phones` arrays are additive: they are populated when the collector's structured extractor finds contact details in the captured page content, and stay empty/absent otherwise. Ignoring them keeps existing behavior unchanged. They do not authorize any auto outreach; the send/act gates (approval required) still apply, and the operator's own credentials/secrets are never read or transmitted.

Do not store unnecessary personal data. Keep safe summaries and source URLs. The human can inspect the original post in their logged-in session when needed.

## Completion Checklist

Before claiming lead/competitor work is complete, verify:

- Stage 10 was loaded.
- Leads include direct and indirect need signals, not only explicit "I need a provider" posts.
- Competitors include direct, indirect, adjacent, attention, or authority competitors when relevant.
- The first lead/competitor pass used 10 scrolls per approved private data source when safe, or documented why it could not.
- Recurring daily runs used 5 scrolls per approved private data source by default, or documented the configured value.
- Detection happened during the same data collection pass unless a human-approved deeper pass was requested.
- Every report opportunity has a post/current URL when available.
- Every report opportunity has a context-aware copy-ready comment.
- Every copy button copies the comment only and does not imply auto-posting.
- Comments are value-first, same-language, short, natural, and not direct ads.
- One or two small natural imperfections are allowed only when they help the comment sound human and do not reduce trust.
- Logs were updated.
- No auto outreach, auto DM, credential/secret collection, or access-control/CAPTCHA bypass occurred. (Collecting publicly-rendered contact details under the operator's command is permitted per the collection-consent rule above; the send/act side still needs human approval.)
