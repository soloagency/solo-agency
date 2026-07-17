# Enrichment etiquette — fair game vs off-limits

Knowing too much about a stranger's private life is not personalization — it reads as creepy and
sinks the email. The line is simple and enforced in code (`crm_store.py enrich write` sends any
`sensitivity: personal` hook to `do_not_mention` and never to email copy).

## `public_business` — fair game (use as hooks)
- New / current / stale listings; listing marketing (photos, video, price changes).
- Work posts, business milestones, awards, "top producer" mentions.
- Public reviews and testimonials about their work.
- Market opinions, neighborhood commentary, professional content they published.
- Brokerage / company / role changes (a business fact, not private).

## `personal` — off-limits in email copy (tag it, never reference it)
- Family, children, relationships, pregnancies, birthdays.
- Health, illness, grief, personal hardship.
- Vacations, home life, hobbies unrelated to the business.
- Anything from a clearly personal (non-business) account or post.

Tag these `sensitivity: personal`. They may inform your *judgment* (e.g. don't email someone who
just posted about a loss) but they go into `do_not_mention`, never into a sentence you send.

## Tone rules for the hook itself (Stage 6 uses these)
- A hook must be a **reason the email exists**, not decoration. Test: delete the personalized
  detail — if the email still stands, the detail was decoration and the reader will smell it.
- Reference the business signal plainly and usefully ("your Main St listing has been up 40 days
  with photo-only — a video tour is closing comparable Alabaster homes in 18 days"), not as
  flattery ("congrats on your amazing listing!").
- One honest, load-bearing detail beats three shallow ones. Never stack personalized details to
  look impressive — it looks scraped.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
