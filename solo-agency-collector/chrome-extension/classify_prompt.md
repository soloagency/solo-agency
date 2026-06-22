# Low-tier classifier for collector feed text

Mục tiêu: cho model rẻ (Claude Haiku / gpt-4o-mini) đọc text feed đã lọc và xuất
ra **một mảng JSON các record** (mỗi post/comment = 1 record). KHÔNG bắt model
dựng cây thread — threading lấy từ URL (`posts/<id>` + `comment_id`) bằng code.

Cách dùng: thay `{{BUSINESS_CONTEXT}}` bằng mô tả doanh nghiệp/niche của bạn, dán
text feed vào `{{FEED_TEXT}}`, gửi cho model với temperature 0.

---

## SYSTEM PROMPT

```
You extract structured records from a social-media GROUP/FEED text dump that was
scraped and pre-cleaned. The text is a flat stream of lines. Links are inline as
path-only URLs like /groups/<gid>/posts/<pid>/?comment_id=<cid> or
/groups/<gid>/user/<uid>/ . Content is mixed Vietnamese/English.

YOUR JOB: output a JSON array. One object per real POST or COMMENT. Judge value
relative to this business context:

BUSINESS CONTEXT: {{BUSINESS_CONTEXT}}

STRUCTURE RULES (the dump is messy — use these cues):
- An author is a SHORT line (a name) that ends in a /user/<id> or /<username>
  link, immediately BEFORE the content line it belongs to.
- A POST body line usually starts with "· Shared with ... group".
- A COMMENT line contains a URL with "comment_id="; its parent post is the
  "/groups/<gid>/posts/<pid>/" part of that same URL -> put it in post_url.
- An engagement line looks like two numbers, e.g. "19 17", often followed by
  "View more answers/comments" -> reactions = first number, comments = second.
  Attach it to the post it follows.
- Author display names that are random letter+digit gibberish (e.g.
  "tenoSprsoduJf1a6084fci1", "DCsQ3O2.comNguyen", "haa3lyu1f0c9") are scraper
  decoys -> set author to "" (empty), do NOT invent a name.

IGNORE (do not emit a record) lines that are pure UI / navigation:
  "sort group feed by New posts", "filter group feed by topic", "All topics",
  standalone "Like Reply See translation", "View more answers/comments/replies",
  "Answer as ...", "Comment as ...", "Top contributor", "See more",
  "See translation", "Rate this translation", "Follow".
Strip those phrases out of the `text` field too; keep only the human content.

For each record output EXACTLY these fields:
- type: "post" | "comment"
- author: string ("" if unknown/decoy)
- profile_url: string (the author's /user/ or /<username> path, "" if none)
- post_url: string (the /.../posts/<id>/ path; for comments take it from
  comment_id URL; "" if none)
- text: string — the cleaned human content, ORIGINAL language, no UI phrases,
  no decoy tokens. Condense lightly but keep meaning, numbers, names, places.
- topic: string — 1-5 words, original or English (e.g. "mua nhà", "solar",
  "mortgage", "remodel", "legal/mold dispute").
- intent: one of "seeking_help" | "sharing_advice" | "advertising_service" |
  "selling" | "announcement" | "social_other"
- role: one of "seeker" | "provider" | "neutral"
  (seeker = needs a product/service; provider = offers/promotes one)
- is_lead: boolean (true if this person is a potential CUSTOMER for the business
  context above)
- is_provider: boolean (true if this person/comment promotes a competing or
  related service/vendor)
- need_or_offer: string — short summary of what they need or what they offer
  (incl. phone/brand/page if present); "" if neither.
- location: string — any place mentioned (state/city); "" if none.
- language: "vi" | "en" | "mixed"
- engagement: {"reactions": int, "comments": int} or null
- relevance: "high" | "medium" | "low" — usefulness as a business signal for the
  context above.

OUTPUT: a single JSON array, nothing else. No markdown, no commentary.
```

## FEW-SHOT (đưa vào system hoặc làm 1 turn mẫu)

INPUT:
```
CharmingHeron3298 /groups/3507238366032465/user/1010666878240464/
· Shared with Private group Anh chị ơi! Em mua nhà được một tháng, mà giờ nhà bị leak nước toàn bộ ở dưới nhà, thêm mold! Trong khi inspection lại k đề cập đến! Giờ e phải làm sao ạ! Ai giới thiệu luật sư BĐS k ạ See translation /
19 17 View more answers /groups/3507238366032465/user/100006095080415/
Khanh H Nguyen /groups/3507238366032465/user/100017389572368/
Top contributor Phải mướn LS. Mà phải chọn loại Business Litigation Attorney. Kiện chủ nhà vì không disclosed, người realtor bán n… See more 3h /groups/nhacuamy/posts/27320378677625100/?comment_id=27320900624239572
QuangTran Vhome /groups/3507238366032465/user/100052405523520/
Verified account Rising contributor Nếu ở Dallas thì Bạn có thể gọi 9799.666.999 or xem thêm các dịch vụ khác của Vhome Construction tại : Facebook.com/vhome.dallas /groups/3507238366032465/user/100064131848214/
```

OUTPUT:
```json
[
  {
    "type": "post",
    "author": "CharmingHeron3298",
    "profile_url": "/groups/3507238366032465/user/1010666878240464/",
    "post_url": "",
    "text": "Em mua nhà được một tháng thì nhà bị leak nước toàn bộ ở dưới + có mold, trong khi inspection không đề cập. Hỏi cách xử lý và xin giới thiệu luật sư bất động sản.",
    "topic": "mold/leak + luật sư BĐS",
    "intent": "seeking_help",
    "role": "seeker",
    "is_lead": true,
    "is_provider": false,
    "need_or_offer": "Cần luật sư bất động sản và hướng xử lý nhà bị leak/mold sau khi mua",
    "location": "",
    "language": "vi",
    "engagement": {"reactions": 19, "comments": 17},
    "relevance": "high"
  },
  {
    "type": "comment",
    "author": "Khanh H Nguyen",
    "profile_url": "/groups/3507238366032465/user/100017389572368/",
    "post_url": "/groups/nhacuamy/posts/27320378677625100/",
    "text": "Phải mướn luật sư, chọn loại Business Litigation Attorney; kiện chủ nhà vì không disclosed và realtor đã bán.",
    "topic": "legal/mold dispute",
    "intent": "sharing_advice",
    "role": "neutral",
    "is_lead": false,
    "is_provider": false,
    "need_or_offer": "",
    "location": "",
    "language": "vi",
    "engagement": null,
    "relevance": "medium"
  },
  {
    "type": "comment",
    "author": "QuangTran Vhome",
    "profile_url": "/groups/3507238366032465/user/100052405523520/",
    "post_url": "",
    "text": "Nếu ở Dallas có thể gọi 9799.666.999 hoặc xem dịch vụ của Vhome Construction tại Facebook.com/vhome.dallas.",
    "topic": "dịch vụ xây dựng",
    "intent": "advertising_service",
    "role": "provider",
    "is_lead": false,
    "is_provider": true,
    "need_or_offer": "Vhome Construction (Dallas) — xây dựng, SĐT 9799.666.999, FB vhome.dallas",
    "location": "Dallas",
    "language": "vi",
    "engagement": null,
    "relevance": "high"
  }
]
```

## USER MESSAGE TEMPLATE

```
Extract records from this feed text:

{{FEED_TEXT}}
```

---

## JSON Schema (cho structured output — gpt-4o-mini `response_format`, hoặc Claude tool)

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
            "language": {"type": "string", "enum": ["vi","en","mixed"]},
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

## Mẹo test
- temperature = 0, đưa cả file (~10-12k token, OK cho 1 lượt).
- Nếu file dài hơn nhiều: chunk ~60-80 dòng/lần để model thấp giữ chính xác cao.
- Sau khi có JSON, **gom theo post bằng code**: group records theo `post_url`
  (hoặc `posts/<id>` trích từ URL) — không để model dựng thread.
