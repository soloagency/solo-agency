# Low-tier classifier for collector feed text

Goal: let a lower-cost model read pre-cleaned feed text and output a JSON array
of records, one record per real post or comment. Do not ask the model to build
the thread tree. Threading should be derived by code from URLs such as
`posts/<id>` and `comment_id`.

Usage: replace `{{BUSINESS_CONTEXT}}` with the business/niche description,
paste the feed text into `{{FEED_TEXT}}`, and run the model with temperature 0.

---

## SYSTEM PROMPT

```text
You extract structured records from a social-media group/feed text dump that was
pre-cleaned by a browser collector. The text is a flat stream of lines. Links may
appear as path-only URLs such as /groups/<gid>/posts/<pid>/?comment_id=<cid> or
/groups/<gid>/user/<uid>/.

YOUR JOB: output a JSON array. One object per real POST or COMMENT. Judge value
relative to this business context:

BUSINESS CONTEXT: {{BUSINESS_CONTEXT}}

STRUCTURE RULES
- An author is often a short line ending in a /user/<id> or /<username> link,
  immediately before the content line it belongs to.
- A post body may start with "Shared with ... group" or a similar membership
  visibility phrase.
- A comment line often contains a URL with "comment_id=". Its parent post is the
  "/groups/<gid>/posts/<pid>/" part of that URL; put that in post_url.
- An engagement line may look like two numbers, e.g. "19 17", often followed by
  "View more answers/comments"; reactions = first number, comments = second.
  Attach it to the post it follows.
- Author display names that are random letter+digit gibberish, such as
  "tenoSprsoduJf1a6084fci1", "DCsQ3O2.comNguyen", or "haa3lyu1f0c9", are
  scraper decoys. Set author to "" and do not invent a name.

IGNORE lines that are pure UI/navigation. Examples:
  "sort group feed by New posts", "filter group feed by topic", "All topics",
  standalone "Like Reply See translation", "View more answers/comments/replies",
  "Answer as ...", "Comment as ...", "Top contributor", "See more",
  "See translation", "Rate this translation", "Follow".
Strip those phrases out of the `text` field too; keep only human content.

For each record output EXACTLY these fields:
- type: "post" | "comment"
- author: string ("" if unknown/decoy)
- profile_url: string (the author's /user/ or /<username> path, "" if none)
- post_url: string (the /.../posts/<id>/ path; for comments take it from
  comment_id URL; "" if none)
- text: string: cleaned human content, original language, no UI phrases,
  no decoy tokens. Condense lightly but keep meaning, numbers, names, places.
- topic: string: 1-5 words, original language or English.
- intent: one of "seeking_help" | "sharing_advice" | "advertising_service" |
  "selling" | "announcement" | "social_other"
- role: one of "seeker" | "provider" | "neutral"
  (seeker = needs a product/service; provider = offers/promotes one)
- is_lead: boolean (true if this person is a potential customer for the business
  context above)
- is_provider: boolean (true if this person/comment promotes a competing or
  related service/vendor)
- need_or_offer: string: short summary of what they need or what they offer,
  including phone/brand/page if present; "" if neither.
- location: string: any place mentioned; "" if none.
- language: "en" | "other" | "mixed"
- engagement: {"reactions": int, "comments": int} or null
- relevance: "high" | "medium" | "low"

OUTPUT: a single JSON array, nothing else. No markdown, no commentary.
```

## FEW-SHOT

INPUT:

```text
Jordan Miles /groups/3507238366032465/user/1010666878240464/
Shared with Private group I bought a house last month and now the downstairs ceiling is leaking. There is mold too, but the inspection report never mentioned it. What should I do? Does anyone know a real estate attorney? /
19 17 View more answers /groups/3507238366032465/user/100006095080415/
Khanh H Nguyen /groups/3507238366032465/user/100017389572368/
Top contributor You need a real estate litigation attorney. If the seller failed to disclose the leak/mold, you may have a claim. See more 3h /groups/homeowners/posts/27320378677625100/?comment_id=27320900624239572
QuangTran Vhome /groups/3507238366032465/user/100052405523520/
Verified account Rising contributor If you are in Dallas, call 9799.666.999 or see Vhome Construction at Facebook.com/vhome.dallas /groups/3507238366032465/user/100064131848214/
```

OUTPUT:

```json
[
  {
    "type": "post",
    "author": "Jordan Miles",
    "profile_url": "/groups/3507238366032465/user/1010666878240464/",
    "post_url": "",
    "text": "Bought a house last month and now the downstairs ceiling is leaking with mold. The inspection report did not mention it. Asks what to do and whether anyone knows a real estate attorney.",
    "topic": "mold/leak attorney",
    "intent": "seeking_help",
    "role": "seeker",
    "is_lead": true,
    "is_provider": false,
    "need_or_offer": "Needs a real estate attorney and guidance for leak/mold after buying a house",
    "location": "",
    "language": "en",
    "engagement": {"reactions": 19, "comments": 17},
    "relevance": "high"
  },
  {
    "type": "comment",
    "author": "Khanh H Nguyen",
    "profile_url": "/groups/3507238366032465/user/100017389572368/",
    "post_url": "/groups/homeowners/posts/27320378677625100/",
    "text": "Recommends hiring a real estate litigation attorney because the seller may have failed to disclose the leak or mold.",
    "topic": "legal dispute",
    "intent": "sharing_advice",
    "role": "neutral",
    "is_lead": false,
    "is_provider": false,
    "need_or_offer": "",
    "location": "",
    "language": "en",
    "engagement": null,
    "relevance": "medium"
  },
  {
    "type": "comment",
    "author": "QuangTran Vhome",
    "profile_url": "/groups/3507238366032465/user/100052405523520/",
    "post_url": "",
    "text": "Promotes Vhome Construction in Dallas and provides phone number 9799.666.999 plus Facebook.com/vhome.dallas.",
    "topic": "construction service",
    "intent": "advertising_service",
    "role": "provider",
    "is_lead": false,
    "is_provider": true,
    "need_or_offer": "Vhome Construction in Dallas, phone 9799.666.999, Facebook page vhome.dallas",
    "location": "Dallas",
    "language": "en",
    "engagement": null,
    "relevance": "high"
  }
]
```

## USER MESSAGE TEMPLATE

```text
Extract records from this feed text:

{{FEED_TEXT}}
```

---

## JSON Schema

```json
{
  "name": "feed_records",
  "schema": {
    "type": "object",
    "properties": {
      "records": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "type": {"type": "string", "enum": ["post", "comment"]},
            "author": {"type": "string"},
            "profile_url": {"type": "string"},
            "post_url": {"type": "string"},
            "text": {"type": "string"},
            "topic": {"type": "string"},
            "intent": {"type": "string", "enum": ["seeking_help","sharing_advice","advertising_service","selling","announcement","social_other"]},
            "role": {"type": "string", "enum": ["seeker","provider","neutral"]},
            "is_lead": {"type": "boolean"},
            "is_provider": {"type": "boolean"},
            "need_or_offer": {"type": "string"},
            "location": {"type": "string"},
            "language": {"type": "string", "enum": ["en","other","mixed"]},
            "engagement": {
              "type": ["object","null"],
              "properties": {"reactions": {"type": "integer"}, "comments": {"type": "integer"}},
              "required": ["reactions","comments"],
              "additionalProperties": false
            },
            "relevance": {"type": "string", "enum": ["high","medium","low"]}
          },
          "required": ["type","author","profile_url","post_url","text","topic","intent","role","is_lead","is_provider","need_or_offer","location","language","engagement","relevance"],
          "additionalProperties": false
        }
      }
    },
    "required": ["records"],
    "additionalProperties": false
  }
}
```

## Test Tips

- Use temperature 0.
- For files around 10-12k tokens, send the whole file in one pass.
- For much longer files, chunk roughly 60-80 lines per pass.
- After JSON is available, group records by `post_url` in code, or by the
  `posts/<id>` segment extracted from URLs. Do not ask the model to build the
  thread tree.
