# PLANS — the Solo Agency plan ladder (rendered copy)

Auto-generated from `plans.json` by `tools/render_plans.py` (run by `deploy-soloagency.sh`). Do not edit by hand —
edit `plans.json` and redeploy. Version 1, updated 2026-09-16.

**Rule for every brain:** never quote a plan number from memory or from any playbook, this file included. When a
number is about to be spoken to the human, run `<bridge> tool plans show --pipeline daily-content-pipeline` (or read
`GET /status` → `entitlement.ladder`) and quote what it returns; the counts come from
`tool crm-store … contact lock-status`, whose `next_tier` names the plan that raises the cap. The server signs the
same ladder into every entitlement token, so the bridge always speaks the server's numbers.

| | Free | Starter $49 | Pro $99 | Business $199 | Enterprise (contact us) |
|---|---|---|---|---|---|
| CRM contacts (stored and unlocked) | 100 | 500 | 2000 | 10000 | unlimited |
| Every data feature: enrich, dossier, contact ladder, harvest incl. people search, Zillow, auto update, priority fixes | yes | yes | yes | yes | yes |
| DM to UNLOCKED contacts (`fb.message.send`) | yes | yes | yes | yes | yes |
| Group post, comment, react (`write_actions`) | no | yes | yes | yes | yes |
| Installs per key (seats) | 1 | 1 | 1 | 1 | 1 |

- `0` / unlimited = no cap. The CRM contact cap is the only sold limit: contacts past it are still captured, just
  locked (no detail view, no email, no DM, no campaign) until the plan grows. `max_clients`, `max_groups`,
  `max_campaigns` and `max_sends_per_day` are flat technical ceilings, not plan differentiators.
- `write_actions` (group post, comment, react) is the only plan-gated feature; it starts at Starter.
- Approaching-cap ratio: `0.8` (`contact lock-status` → `approaching: true` once `unlocked / max_contacts` reaches it).
- Upgrade path: https://widecast.ai/#setup (the Solo Agency tier follows the WideCast plan on that account).

Ladder in one line: Free 100 · Starter $49 → 500 · Pro $99 → 2000 · Business $199 → 10000 · Enterprise (contact us) → unlimited.
