# PLANS — the Solo Agency plan ladder (rendered copy)

Auto-generated from `plans.json` by `<bridge> tool plans render` (run by `deploy-soloagency.sh`). Do not edit by hand —
edit `plans.json` and redeploy. Version 2, updated 2026-09-17.

**Rule for every brain:** never quote a plan number from memory or from any playbook, this file included. When a
number is about to be spoken to the human, run `<bridge> tool plans show --pipeline daily-content-pipeline` (or read
`GET /status` → `entitlement.ladder`) and quote what it returns; the counts come from
`tool crm-store … contact lock-status`, whose `next_tier` names the plan that raises the cap. The server signs the
same ladder into every entitlement token, so the bridge always speaks the server's numbers.

| | Free | Starter $49 | Pro $99 | Business $199 | Enterprise (contact us) |
|---|---|---|---|---|---|
| CRM contacts (stored and unlocked) | 100 | 500 | 2000 | 10000 | unlimited |
| Every data feature: enrich, dossier, contact ladder, harvest incl. people search, Zillow, auto update, priority fixes | yes | yes | yes | yes | yes |
| Group posts per day (`fb.group.post`, `fb.profile.post`, `x.post.publish`) | 1 | unlimited | unlimited | unlimited | unlimited |
| Comments and replies per day (`fb.post.comment`, `x.post.reply`, `ig.post.comment`) | 3 | unlimited | unlimited | unlimited | unlimited |
| Direct messages per day, to UNLOCKED contacts (`fb.message.send`, `ig.message.send`) | 3 | unlimited | unlimited | unlimited | unlimited |
| Installs per key (seats) | 1 | 1 | 1 | 1 | 1 |

- `0` / unlimited = no cap. Two things are sold: the CRM contact cap — contacts past it are still captured, just
  locked (no detail view, no email, no DM, no campaign) until the plan grows — and the daily write allowance:
  group posts, comments/replies and direct messages are counted per install per local day, every client together.
  Free gets 1 post, 3 comments or replies, 3 direct messages per day to prove the lanes work; every paid tier is uncounted. An approved item past the day's
  allowance waits at the back of the queue for the next day (the Approval page and the report say so), or goes
  sooner after an upgrade. The safety caps in system settings still apply on top. `max_clients`, `max_groups`,
  `max_campaigns` and `max_sends_per_day` are flat technical ceilings, not plan differentiators.
- Every tier carries `write_actions`; on a tier with a write allowance the write capabilities run only through the
  Approval page queue (`write_via_approval_only` cuts them from any other job), so the count cannot be bypassed.
- Approaching-cap ratio: `0.8` (`contact lock-status` → `approaching: true` once `unlocked / max_contacts` reaches it).
- Upgrade path: https://widecast.ai/#setup (the Solo Agency tier follows the WideCast plan on that account).

Ladder in one line: Free 100 · Starter $49 → 500 · Pro $99 → 2000 · Business $199 → 10000 · Enterprise (contact us) → unlimited.
