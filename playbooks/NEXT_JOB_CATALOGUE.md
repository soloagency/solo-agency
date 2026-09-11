# Next Job Catalogue — what the Team Leader offers when nothing is waiting on it

This file is loaded at the start of every session together with `playbooks/TEAM_MODEL.md` (see
that file, "The Team Leader's standing duties"). It is the concrete instrument behind the
Next-Action Guidance Rule (`playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`, Response Self-Audit
Checklist): instead of a generic "here are some next steps," the Team Leader runs one fixed
STATE POLL, ranks whatever it finds by the priority tiers below, and reads the human-facing offer
straight off the CATALOGUE row that matched. Approved by the owner, 2026-09-09 night.

## Purpose

A Boss-facing reply never ends with nothing to do and never ends with an invented action either.
This file gives the Team Leader:

1. one poll to run before composing any reply (the STATE POLL);
2. a fixed order for what matters most when several things are true at once (the priority tiers,
   plus the IDLE RULE for when nothing is pending);
3. the exact shape of the block that offers work (the "next jobs" block); and
4. a CATALOGUE of every job the Team Leader knows how to offer, each tied to the machine-readable
   condition that makes it relevant right now — never a menu recited from memory, never a fixed
   weekly nag.

This is separate from `playbooks/FEATURE_CATALOG.md` (the Feature Discovery Rule: introducing a
product capability the human has not tried) and separate from the PRIMING vs SELLING plan
messaging owned by `AGENTS.md` / `playbooks/10_LEAD_COMPETITOR_DETECTION.md`. This file answers
one narrower question: *given what actually happened in this install, what is the next concrete
piece of work worth offering, in what order.*

## The LANGUAGE RULE

Every offer, priming line, and upsell line is spoken in the language the human is using in the
session, worded the way a real person would offer work — never a fixed template pasted verbatim.
The Vietnamese and English lines in this file (the `example VI` / `example EN` columns and the
Worked examples below) are EXAMPLES OF TONE ONLY, not scripts to output character-for-character.
Numbers inside an offer (lead counts, locked counts, days stale, N contacts) always come from the
STATE POLL's real values — never estimated, never rounded to sound better, never carried over from
a previous run without re-reading the source.

## The STATE POLL

Run this before composing any Boss-facing reply, scheduled-run reply, or INTERNAL_REPORT "Next
Operator Action" line. Each signal names the exact file, tool, or field it reads — never a hand
count, never a guess from memory:

| # | Signal | Read from |
|---|---|---|
| 1 | Entitlement / plan state | `GET /status` → `entitlement` (`tier`, `mode`, `reason`); `tool entitlement status` |
| 2 | Contact lock state | `<bridge> tool crm-store --client-dir <CLIENT_DIR> contact lock-status` → `max_contacts`, `lockable`, `unlocked`, `locked`, `upgrade_url` |
| 3 | Pending outreach approvals | count of files under every `campaigns/{slug}/outbox/pending_approval/` |
| 4 | Pending content approvals | the run's Approval Workflow queue (`playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`, "23.3 Approval Workflow") |
| 5 | Discovery shortlist awaiting promotion | `history/YYYY-MM/facebook_discovery_shortlist.jsonl` rows with `status: recommended` |
| 6 | Standup backlog | `daily-content-pipeline/automation/standup.jsonl` tail — `needs_boss[]`, `blockers[]` per line |
| 7 | Boss-orders ledger | `daily-content-pipeline/automation/boss_orders.md` rows with `status` in `waiting_boss`, `blocked` |
| 8 | Sendbox health | `sendboxes/sendboxes.json` — any entry `status: needs_reauth`, or none `status: healthy` |
| 9 | Campaign roster and status | each client's `campaigns/{slug}/campaign_config.json` — `channel_strategy`, `status` (`active`/`paused`) |
| 10 | Social lead source (per platform) | Client Intelligence Profile `facebook_lead_source` / `instagram_lead_source` / `x_lead_source`, each `enabled\|web_only\|pending` |
| 11 | Private data source state | `private_data_source_discovery.status` (`not_asked\|recommended\|declined\|postponed\|partially_approved\|approved\|pending_human_approval\|pending_private_activation\|active\|blocked\|completed\|discovery_declined_or_postponed`); per item in `private_data_sources.items[]` — `approval_status` (`pending_human_approval\|approved\|rejected`), `activation_status` (`pending_private_activation\|active\|declined_for_now\|unavailable`) |
| 12 | Extension / collector health | `extension_health.status` (`recent\|stale\|no_extension_check_yet`, or `unavailable` when the bridge itself cannot be reached) past the 75-second grace window |
| 13 | Last run recency | most recent `standup.jsonl` line's `ts` for this client, and the corresponding `fleet/{client_slug}.json.report.last_report_at` |
| 14 | Published content / analytics staleness | `fleet/{client_slug}.json` → `engagement`, `report`; last analytics pull date |
| 15 | Provider (WideCast) connection | `integrations/providers/provider_config.local.json` + `provider_capabilities.json` verified identity |
| 16 | Discovered sources awaiting a decision | source registry rows `kind: discovered` — count `status: new` (awaiting the Boss's approve/dismiss) and count `status: approved` (approved, not yet harvested) (`playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`) |

Read every row that is cheap to read (files already on disk) on every reply; skip a row only when
its file genuinely does not exist yet for this install (e.g. no CRM workspace, no campaigns) and
say so rather than inventing a zero.

## Priority tiers and the IDLE RULE

When more than one signal fires, resolve in this order — always show the highest tier that fired,
never bury it under a lower one:

1. **Backlog** — approvals waiting (signals 3-4), a recommended discovery shortlist (signal 5),
   discovered sources with `status: new` (signal 16), an unresolved blocker (signal 6), or a
   Boss-orders row stuck on `waiting_boss`/`blocked` (signal 7). Nothing else gets offered ahead of
   clearing what is already sitting there.
2. **Lead generation** — running scans/harvests against sources already approved, to bring in more
   leads (Social Discovery Pass, private-group scans, friend/people/Zillow harvest, list import).
3. **Lead exploitation** — working leads already in the CRM (hello campaigns, enrichment,
   follow-up bumps, comment/group-post campaigns, pipeline review, analytics).
4. **Expansion** — growing the surface itself: new sources, new discovery terms/cities,
   new competitors to track, a new client, a new content format, a new notification channel.
5. **Plan** — an upgrade offer. Only reached with a real locked count (`contact lock-status` →
   `locked > 0`) or a write action a Free install actually needed and was refused. Never offered
   as a filler when tiers 1-4 found nothing — a plan pitch is never busywork.

**IDLE RULE.** If the poll finds nothing pending in tiers 1-3 and no new leads landed this run, the
Team Leader does not wait silently for orders. It proposes two concrete jobs from tier 2 or tier 4
— something that would generate more leads or grow the surface — and asks which one to take. An
idle install still gets offered real work; it never gets "let me know if you need anything."

## The meter

Signal #2 (`contact lock-status`) drives one more thing beyond the tier-5 upgrade offer: a
standing line that travels with every Boss-facing reply once it turns on, independent of whether
this reply's next-jobs block or ACTION REQUIRED block is showing.

- **Locked.** Whenever `contact lock-status` shows `locked > 0`, every Boss-facing reply,
  INTERNAL_REPORT, and morning brief carries the one-line meter, rendered in the human's language,
  no cooldown:

  ```text
  {L} leads locked under {tier} — {unlocked}/{max} open
  ```

- **Approaching the cap.** When `locked == 0` and `unlocked / max_contacts >= 0.8`, carry the
  approaching-cap line in the same place instead:

  ```text
  {unlocked}/{max} open contacts used; new leads may start locking
  ```

- The ratio and both counts come straight from `contact lock-status`'s `unlocked`/`max_contacts`/
  `locked` fields — never computed from a CRM list count or a report total. The meter is not an
  upsell moment by itself (see Guardrails): it is a standing fact, shown the same way every reply,
  and it is what lets `review_locked_leads_upgrade` fire once and then step back — the meter keeps
  the number visible without repeating the pitch.

## The "next jobs" block

Every Boss-facing reply, scheduled-run reply, and INTERNAL_REPORT "Next Operator Action" line ends
with EITHER the `**[ACTION REQUIRED]**` block(s) already required elsewhere in the playbooks, OR
this next-jobs block — never both stacked as filler, never neither. The block is 2-3 offers picked
by the STATE POLL and the priority tiers above, plus exactly one closing question:

```text
1. "{one line the human could literally say back to start this job}" — needs: {the one prerequisite, or "nothing, ready now"}.
2. "{second offer, same shape}" — needs: {...}.
3. "{third offer, optional}" — needs: {...}.

{exactly one closing question, in the human's language}
```

Never "No action required right now," never a passive sign-off, never more than three offers, and
never an offer whose `needs` prerequisite is unmet without saying so plainly (an offer that needs a
sendbox first is shown as needing one, not presented as one-step).

## The CATALOGUE

`tier` is the priority tier from above. `signal` is the machine-readable condition that makes the
offer relevant now. `plan` is the minimum plan the job itself needs (write actions start at
Starter; everything else is a data feature and runs on every plan, Free included — `AGENTS.md`,
"Plans"). Owner's own examples from the 2026-09-09 design session are marked with `*` after the id.

| id | tier | signal | offer | example VI | example EN | runs | needs | plan |
|---|---|---|---|---|---|---|---|---|
| `show_approval_report` * | 1 | pending-approval files > 0 (poll #3-4) | review and approve/reject what's waiting | "Anh có {N} email/bài đang chờ duyệt, em trình luôn cho anh xem không?" | "You have {N} emails/posts waiting for your approval — want me to show them now?" | Approval Workflow (`outreach/playbooks/00_CORE_CONTEXT_REQUIREMENTS.md` step 5-6; `09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` §23.3) | nothing, ready now | Free+ |
| `promote_discovered_groups` | 1 | shortlist rows `status: recommended` (poll #5) | promote or reject the recommended groups | "Em tìm được {N} nhóm Facebook hợp, anh duyệt nhóm nào để em theo dõi mỗi ngày?" | "I found {N} Facebook groups that fit — which ones should I start monitoring daily?" | `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, Social Discovery Pass (Facebook leg — shortlist stays Facebook-only) | `facebook_lead_source: enabled` | Free+ |
| `review_discovered_sources` | 1 | discovered-source rows `status: new` (poll #16) | list the new discovered threads with excerpt and reason, read a post aloud on request | "Em tìm được {N} bài mà người trả lời có thể là khách của anh — anh nghe qua từng bài không?" | "I found {N} posts whose repliers might be your buyers — want me to walk through them?" | `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Step 5" | nothing, ready now | Free+ |
| `harvest_thread` | 2 | Boss names a discovered source (or approves one via the Discovered tab) (poll #16, `status: new`/`approved`) | run the batch triage harvest on that one thread now | "Em quét bình luận bài đó và đưa những người phù hợp vào CRM nhé?" | "Want me to triage that thread's comments and add the right people to the CRM?" | `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Harvest a discovered thread (on the Boss's order only)" | a named `source_id`, `status: new` or `approved` | Free+ |
| `resolve_blocker` | 1 | `standup.jsonl` tail `blockers[]` non-empty (poll #6) | walk through the blocker and fix it now | "Lần chạy gần nhất bị vướng {blocker}, anh muốn em xử lý ngay không?" | "The last run hit a blocker ({blocker}) — want me to work through it now?" | whichever stage owns the blocker code | nothing, ready now | Free+ |
| `answer_boss_orders` | 1 | ledger row `waiting_boss`/`blocked` (poll #7) | surface the row, ask the missing question | "Đơn hàng \"{order}\" đang chờ anh trả lời {question}." | "Your order \"{order}\" is waiting on your answer to {question}." | `playbooks/TEAM_MODEL.md`, Boss-orders ledger | nothing, ready now | Free+ |
| `social_login_reconnect` (formerly `fb_login_reconnect`) | 1 | `extension_health.status` `stale`/`no_extension_check_yet` past grace (poll #12), or any of `facebook_lead_source`/`instagram_lead_source`/`x_lead_source` `web_only` with reason `not logged in on {date}` (poll #10) | reconnect the login(s) — names whichever platform(s) are logged out/stale | Login Reminder block, verbatim, naming the platform(s) affected | Login Reminder block, verbatim, naming the platform(s) affected | `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 12D; `playbooks/SETUP_FLOW_ENTRYPOINT.md`, "Kết nối Facebook, Instagram and X (step 4)" | Chrome extension reload/login for the named platform(s) | Free+ |
| `social_web_only_upsell` (formerly `fb_web_only_upsell`) | 2 | parameter: `platform` — any of `facebook_lead_source`/`instagram_lead_source`/`x_lead_source` `: web_only` from the human's own deliberate choice, not the auto logged-out path (poll #10) | offer to turn that platform back on for more leads | "Bật lại {platform} để tăng lead nhé? Vòng quét đầu tiên sau khi bật dùng ngân sách khám phá lần đầu ({platform} First Run: Facebook tối đa 21 lượt gọi / Instagram, X tối đa 12 lượt gọi mỗi bên, trải trong ≥ 4 giờ)." | "Want to turn {platform} back on for more leads? The first scan after that uses the First Run discovery budget ({platform} First Run: Facebook up to 21 collector calls / Instagram and X up to 12 each, spread over 4+ hours)." | `playbooks/SETUP_FLOW_ENTRYPOINT.md`, "Kết nối Facebook, Instagram and X (step 4)"; `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, Social Discovery Pass | `{platform}_lead_source: web_only`; on acceptance, follow the 90-second extension help loop and the same acknowledgment/recording contract as setup step 4 | Free+ |
| `collector_healthcheck` | 1 | last healthcheck > 24h old, or file-queue path looks stale | run a healthcheck pass now | "Em chạy kiểm tra sức khoẻ hệ thống thu thập ngay không?" | "Want me to run a collector healthcheck right now?" | `playbooks/HEALTHCHECK.md` | nothing, ready now | Free+ |
| `update_watch_setup` | 1 | GitHub `main` ahead of local commit/bridge version, or update-watch task missing from `automation_manifest.md` | apply the update, or set up the watch task | "Có bản cập nhật mới — em áp dụng luôn không?" | "There's a newer version available — want me to apply it now?" | `playbooks/11_UPDATE_AND_VERSION_WATCH.md` | Boss approval unless `auto_apply_approved: true` | Free+ |
| `post_support_group` | 1 | confirmed bug/question with no matching open row in `support_requests.md` | draft the support post, ask approval | draft shown per `TEAM_MODEL.md` post template | draft shown per `TEAM_MODEL.md` post template | `playbooks/TEAM_MODEL.md`, "Support requests" | Boss approval of exact text | Free+ |
| `social_discovery_pass` (formerly `fb_discovery_pass`) | 2 | parameter: `platform` — offered per platform whose last pass is older than 7 days (Facebook: also offered when its shortlist is stale) | run today's Social Discovery Pass now for that platform | "Em chạy một vòng Social Discovery Pass cho {platform} ngay bây giờ nhé?" | "Want me to run a Social Discovery Pass for {platform} right now?" | `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, Social Discovery Pass | `{platform}_lead_source: enabled`, extension healthy | Free+ |
| `scan_private_groups` * | 2 | ≥1 item in `private_data_sources.items[]` with `approval_status: approved` and `activation_status: active`, next scheduled scan > 24h away, or Boss asks out-of-cycle | run an extra private-group scan now | "Anh muốn em quét thêm nhóm riêng ngay bây giờ không, ngoài lịch hằng ngày?" | "Want an extra private-group scan right now, outside the daily schedule?" | `playbooks/02_PRIVATE_SOURCE_SETUP.md`; `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` | ≥1 `private_data_sources.items[]` entry with `approval_status: approved`, `activation_status: active`; Local Collector healthy | Free+ |
| `harvest_friend_list` * | 2 | no active `channel_strategy: friend_harvest` campaign, or seeds fully walked | start/extend a friend-list harvest | "Em bắt đầu quét danh sách bạn bè của {seed} để tìm khách tiềm năng nhé?" | "Want me to start mining {seed}'s friend list for prospects?" | `outreach/playbooks/16_FRIEND_HARVEST.md` | a seed profile named by the Boss | Free+ |
| `persona_people_hunt` | 2 | Boss names a persona/target not covered by an existing discovery term | run a targeted people-search pass | "Em chạy tìm người theo đúng mô tả đó (\"{persona}\") ngay không?" | "Want me to run a targeted people-search for \"{persona}\" right now?" | `fb.people.search` (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`) | the persona description | Free+ |
| `mine_network_ff` | 2 | a seed's friend list already harvested once; a friend-of-friend pass would find more | widen the harvest to a second-degree seed | "Muốn em mở rộng qua bạn của bạn từ {seed} để tìm thêm khách không?" | "Want me to widen the harvest to {seed}'s friends-of-friends?" | `outreach/playbooks/16_FRIEND_HARVEST.md` (additional `seed_profiles`) | a second-degree seed | Free+ |
| `zillow_harvest` | 2 | real-estate vertical, no `zillow.agents.list` pull in 7 days | pull a fresh batch of Zillow agent profiles | "Em kéo danh sách agent Zillow mới cho khu vực này nhé?" | "Want me to pull a fresh batch of Zillow agent profiles for this area?" | `outreach/playbooks/17_ZILLOW_HARVEST.md` | human gate (chime, Press & Hold) | Free+ |
| `import_list` | 2 | Boss mentions an existing list not yet in the CRM | import the list into the CRM | "Anh gửi file danh sách đó, em nhập vào CRM luôn nhé?" | "Send me that list and I'll import it straight into the CRM?" | `outreach/playbooks/03_IMPORT_LIST.md` | the file from the Boss | Free+ |
| `hello_campaign_existing_crm` * | 3 | unlocked CRM contacts at `lead` stage never emailed | start a first-touch hello campaign | "CRM đã có {N} liên hệ chưa từng nhận mail — em soạn chiến dịch chào hỏi đầu tiên nhé?" | "The CRM has {N} contacts who've never been emailed — want me to draft a first hello campaign?" | `outreach/playbooks/05_CAMPAIGN_MANAGEMENT.md`; `06_EMAIL_WRITING_STANDARD.md` | a healthy sendbox | Free+ |
| `connect_gmail_sendbox` | 3 | no `sendboxes.json` entry `status: healthy` | connect or repair a Gmail sendbox | "Em cần một hộp Gmail đang hoạt động để gửi mail — anh kết nối giúp em không?" | "I need a working Gmail sendbox to send — can you connect one?" | `outreach/playbooks/02_SENDBOX_SETUP.md` | Boss's Gmail, via the UI connect flow | Free+ |
| `enrich_leads` | 3 | unlocked contacts with enrichment pending | enrich leads before outreach | "Em làm giàu thông tin cho {N} liên hệ trước khi gửi mail nhé?" | "Want me to enrich {N} contacts before we reach out?" | `outreach/playbooks/04_VERIFY_ENRICH.md` | unlocked contacts pending enrichment | Free+ |
| `followup_bump_check` | 3 | contacts returned by `tool crm-store … followups due` past the follow-up window | draft a value-add follow-up bump | "Có {N} liên hệ chưa trả lời quá hạn, em soạn follow-up nhẹ nhàng không?" | "{N} contacts have gone quiet past the follow-up window — want a bump drafted?" | `outreach/playbooks/10_FOLLOWUP_REPLY_MANAGEMENT.md` | an existing sent step | Free+ |
| `show_pipeline` | 3 | Boss hasn't reviewed the pipeline this session and deals exist across stages | show the current pipeline snapshot | "Em cho anh xem nhanh pipeline hiện tại không?" | "Want a quick look at the current pipeline?" | `outreach/playbooks/14_TASKS_TODAY_VIEW.md` | nothing, ready now | Free+ |
| `comment_campaign` * | 3 | commentable posts detected (author passed the Lead Qualification Rule at hot/warm/watch AND the post's topic fits `goal.description`), no active comment campaign | Starter+: start commenting; Free: preview only | Starter: "Em có {N} bài đáng để bình luận, bắt đầu chiến dịch comment không?" · Free: "Em có {N} bài đáng bình luận — xem trước danh sách, nâng Starter để em tự bình luận giúp anh." | Starter: "I found {N} posts worth commenting on — start a comment campaign?" · Free: "I found {N} posts worth commenting on — want a preview? Starter lets me comment for you." | `outreach/playbooks/18_COMMENT_CAMPAIGN.md` | `write_actions` for the live action | Starter+ (Free = preview) |
| `group_post_campaign` * | 3 | approved content ready, no active group-post campaign | start posting into the target groups | "Nội dung đã sẵn sàng — em bắt đầu đăng vào nhóm giúp anh không?" | "Content's ready — want me to start posting it into the groups?" | `outreach/playbooks/19_GROUP_POST_CAMPAIGN.md` | `write_actions`, approved content | Starter+ |
| `review_analytics` | 3 | no analytics pull in > 7 days on published content | pull the latest analytics, show what's working | "Đã lâu chưa xem số liệu — em kéo báo cáo hiệu quả mới nhất không?" | "It's been a while since we checked performance — want the latest analytics pulled?" | `playbooks/05_MEASURE_LEARN_IMPROVE.md` | published content + provider connection | Free+ |
| `add_private_sources` | 4 | `private_data_source_discovery.status` in `not_asked\|declined\|postponed` with no items yet in `private_data_sources.items[]`, or few `activation_status: active` items | add more private groups/pages to watch | "Anh có nhóm/trang riêng nào khác muốn em thêm vào danh sách theo dõi không?" | "Any other private groups or pages you'd like me to start watching?" | `playbooks/02_PRIVATE_SOURCE_SETUP.md` | Local Collector + extension healthy | Free+ |
| `expand_discovery_terms` | 4 | Boss names a new city/niche absent from the keyword bank | add a discovery-term set for that city/niche | "Em thêm bộ từ khoá cho khu vực/ngành mới đó vào để bắt đầu quét không?" | "Want me to add a keyword set for that new city/niche and start scanning it?" | keyword bank (`playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`); `10_LEAD_COMPETITOR_DETECTION.md` | the new city/niche named | Free+ |
| `track_competitor` | 4 | Boss names a competitor not yet on the watch list | start tracking a named competitor | "Em thêm {competitor} vào danh sách theo dõi đối thủ nhé?" | "Want me to start tracking {competitor} as a competitor?" | `playbooks/10_LEAD_COMPETITOR_DETECTION.md` | the competitor's name/URL | Free+ |
| `make_video` | 4 | an approved idea/script has no video produced yet | produce the finished video for that idea | "Ý tưởng đã duyệt rồi — em dựng video luôn không?" | "That idea's approved — want me to produce the video now?" | `playbooks/skills/video-script-writing`, `playbooks/skills/video-editing` | PDNA/WideCast provider connected | Free (create); publish spends WideCast credits |
| `write_blog_social` | 4 | an approved idea has no blog/social draft yet | draft the blog post and social captions | "Em viết bài blog và caption mạng xã hội cho ý tưởng đó nhé?" | "Want me to draft the blog post and social captions for that idea?" | `playbooks/skills/blog-writing`, `playbooks/skills/social-post-writing` | an approved idea | Free+ |
| `connect_notification` | 4 | `notification_channel: local_path_only` recorded | connect Telegram/email/WideCast notification | "Hiện tại báo cáo chỉ lưu máy — anh kết nối Telegram hoặc email để em gửi thẳng cho anh không?" | "Reports only save locally right now — connect Telegram or email so I can send them to you?" | `playbooks/04_DAILY_SCHEDULE.md`, notification channel setup | a Telegram/email/WideCast account | Free+ |
| `add_client` | 4 | Boss mentions a new business/client not in `clients_index.md` | onboard the new client | "Em thêm khách hàng mới đó vào hệ thống luôn nhé?" | "Want me to onboard that new client now?" | `09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` §13, Incremental Client Onboarding Rule | basic profile info from the Boss | Free+ (no per-client plan limit) |
| `morning_brief_setup` | 4 | no recurring morning brief configured, or Boss asks for one every day | set up/confirm the daily morning brief | "Em thiết lập báo cáo buổi sáng gửi đều mỗi ngày cho anh nhé?" | "Want me to set up a daily morning brief for you?" | `playbooks/TEAM_MODEL.md`, "Standup"; `SCHEDULED_RUN_ENTRYPOINT.md` step 16A | a notification channel connected | Free+ |
| `review_locked_leads_upgrade` | 5 | `contact lock-status` → `locked > 0`, first time this session | ACTION REQUIRED upgrade offer, not a "next jobs" line | "Em vừa đưa {N} lead mới vào CRM, {H} lead có tín hiệu tốt. Gói {tier} đang mở {unlocked} contact, {locked} lead còn lại đang khoá chi tiết, chưa gửi mail hay nhắn tin được. Mở gói {next_tier} thì {next_cap} contact mở ngay, không cần quét lại." | "I just brought {N} new leads into the CRM, {H} of them look promising. {tier} keeps {unlocked} contacts open — the other {locked} are locked (no detail, no email, no DM). Upgrading to {next_tier} opens {next_cap} contacts immediately, no rescan needed." | `AGENTS.md`, "Plans"; `https://widecast.ai/#setup` | `locked > 0` | upgrade offer |
| `write_actions_upgrade` | 5 | a Free install needed `fb.group.post`/`fb.profile.post`/`fb.post.comment`/`fb.post.react` and was refused | ACTION REQUIRED upgrade to unlock write actions | "Việc này cần đăng bài/bình luận/react hộ anh — tính năng này mở từ gói Starter. Anh nâng gói tại {upgrade_url} để em làm luôn." | "This needs posting/commenting/reacting on your behalf — that starts at Starter. Upgrade at {upgrade_url} and I'll do it right away." | `AGENTS.md`, `write_actions` gate | the refusal already happened | Starter+ |

### Notes on specific rows

- **Any URL this catalogue hands to the human** — the local dashboard/CRM link (`http://127.0.0.1:17321/ui/...`) as well as an external one like `review_locked_leads_upgrade`'s/`write_actions_upgrade`'s upgrade URL — follows the SHOW RULE (`docs/UI_DESIGN.md` §1 principle 2, OWNER DECISION 2026-09-10): print it as text every time; on Claude Code desktop also open it in the side Browser pane; on any other local runtime also run `open`/`start`/`xdg-open` so it lands in a real browser; never HTTP-GET it to "verify." The local dashboard opens directly now (`--ui-auth host` default) — no entry-link/token step, no "Locked" page. A job offer that shows data — a lead count, a report, a campaign's state, the locked-contacts meter — also follows the Answer-and-Show Rule (`SOLO_AGENCY_PLAYBOOK.md`, "Team Leader Reply Frame"): the chat line stands alone, and the dashboard navigation is the matching routes-table view for depth.
- **`comment_campaign`** is the one row with a plan-dependent offer shape, not a plan-dependent
  signal: the signal (commentable posts detected) fires the same on every plan — "commentable"
  means the post's author passed `playbooks/LEAD_QUALIFICATION_RULE.md` at `hot`, `warm`, or `watch`
  (the first gate, run against the client's `buyer_profile`) AND the post's topic fits
  `goal.description` (the second gate, the topical/voice check); a `competitor`/`none` decision or a
  topic mismatch does not count toward the signal. On Free, the
  catalogue's job is to show the preview — the list of posts worth commenting on — never the
  action itself; the upgrade mention here is a value-first aside inside the offer line, not a
  separate `write_actions_upgrade` block, unless the Boss explicitly asks to start commenting and
  is then refused (that refusal is what triggers `write_actions_upgrade`).
- **`zillow_harvest`** always keeps the human-in-the-loop gate from `ZILLOW_CAPABILITIES.md` (a
  chime, then the operator does the Press & Hold) — the offer in chat is to start the pull, never
  a claim that it completes unattended.
- **`social_login_reconnect`** never freelances new wording: it reuses the Login Reminder block
  quoted verbatim in `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 12D and in the setup roadmap's
  step 4 ("Kết nối Facebook, Instagram and X", `playbooks/SETUP_FLOW_ENTRYPOINT.md`) — the same
  block, just fired again later when a session goes stale, and naming only the platform(s) actually
  affected (funnel moment B carries a second sentence after that verbatim block — this catalogue
  does not duplicate or paraphrase the block itself). This is distinct from `social_web_only_upsell`:
  `social_login_reconnect` fires when a platform that was `enabled` goes stale/logged-out (including
  the automatic logged-out `web_only` set by the run itself — `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`,
  "Auto web_only on logged-out"); `social_web_only_upsell` fires when a platform is `web_only` from
  the human's own deliberate, acknowledged choice — never a stale session — and its acceptance runs
  the same 90-second extension help loop and acknowledgment/recording contract as setup step 4, not
  a bare reconnect. Future, not yet built (pending): once the extension reports
  `{platform}_logged_in: true|false` at check-in per platform, this same signal will be able to tell
  a genuinely stale/logged-out extension apart from one that is simply connected to the wrong
  browser profile — reference that only as "when the extension reports `{platform}_logged_in`,"
  never as a state that exists today. Every run already re-probes a `web_only`(not logged in)
  platform on its own, with its step-1 call in its normal round-robin slot
  (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Re-probe on every run") — so this job is never the
  prerequisite for that platform's recovery. It is an offer to log in now, sooner than waiting for
  the next automatic probe to catch it.
- **`social_web_only_upsell`** never assumes the human forgot; it offers, does not pressure, and
  always names the FIRST RUN discovery budget that will apply to that platform's first Social
  Discovery Pass leg once they enable it — because a `web_only` client who later enables a platform
  still gets that platform's FIRST RUN budget on whichever run turns out to be its first-ever pass,
  per `{platform}_discovery_first_pass_done`.
- **`review_discovered_sources`** and **`harvest_thread`** are distinct: the first only lists what
  Step 5 already recorded (read-only, `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Step 5") and
  reads a post's text aloud on request — it never triages comments or writes to the CRM; the second
  is the actual harvest job and only runs against a source the Boss named (or approved via the
  Discovered tab), never against "all new ones" in one call. Answer-and-Show for both opens
  `/ui/{client}/sources?tab=discovered`.
- **`mine_network_ff`** is distinct from `harvest_friend_list`: the first offer starts a harvest
  from a Boss-named seed's own friend list; this one only becomes relevant after that harvest has
  already run once and widening to a friend-of-friend seed is the next lever — it is never offered
  before a first `harvest_friend_list` pass exists for this client.
- **`review_locked_leads_upgrade`** and **`write_actions_upgrade`** are the only two rows that
  produce an `**[ACTION REQUIRED]**` block instead of a next-jobs line — see "The meter" above for
  how the locked count keeps being shown after the one-time ACTION REQUIRED fires.

## Relationship to other rules

This file does not replace or loosen any existing hard gate — it decides which already-legal
action to name and in what order:

- The Next-Action Guidance Rule and the `**[ACTION REQUIRED]**` block conventions
  (`09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`) still govern the mechanics of how a required action or
  a next-step question is formatted; this file supplies the content and the priority order that
  feed into them.
- `playbooks/FEATURE_CATALOG.md`'s Feature Discovery Rule is a separate, narrower mechanism (value-
  first introduction of an unused product capability at setup-complete, a no-pending-action
  handoff, a lead-detected run, or the weekly cadence) and keeps its own once-per-moment rotation;
  it is not folded into the next-jobs block and the two are not offered as duplicates in the same
  reply.
- The PRIMING vs SELLING rule and the FUNNEL MOMENTS (A-H) named in the monetization design stay
  exactly as scoped elsewhere (`AGENTS.md`; `playbooks/10_LEAD_COMPETITOR_DETECTION.md`,
  "Capture never stops at the plan's contact cap"; `playbooks/02_PRIVATE_SOURCE_SETUP.md`;
  `playbooks/SETUP_FLOW_ENTRYPOINT.md`; `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md`) — this file
  only adds the two tier-5 CATALOGUE rows (`review_locked_leads_upgrade`, `write_actions_upgrade`)
  and the meter definition so they sit inside the same STATE POLL as everything else, instead of
  living as a separate, uncoordinated check.
- **Capability-named rows are not missing playbooks.** `persona_people_hunt` (`fb.people.search`),
  `mine_network_ff` (the same `outreach/playbooks/16_FRIEND_HARVEST.md` pass as
  `harvest_friend_list`, run again with a second-degree seed), `track_competitor`, and
  `expand_discovery_terms` (the keyword bank) name a capability id or a mechanism/playbook already
  dedicated to a broader job in the `runs` column, instead of a dedicated file of their own — they
  run through that named skill/stage, never an invented one. No row in this CATALOGUE cites a
  `playbooks/…` or `outreach/playbooks/…` path that does not exist on disk; before adding a new row,
  verify with `grep -oE '(playbooks|outreach/playbooks)/[A-Za-z0-9_./-]+'
  playbooks/NEXT_JOB_CATALOGUE.md | sort -u`, then `ls` every result.

## Worked examples

**A. Fresh install, first run just finished.** The First-Run Report (`SOLO_AGENCY_PLAYBOOK.md`,
"Wait and report") precedes these offers — it states when the run finished, leads found, the
locked-contacts meter, and what needs the Boss, before this poll's offers are ever spoken. Poll:
signal 2 → `locked: 0`, `unlocked: 12`,
`max_contacts: 30` (well under 0.8, no meter). Signal 5 → 6 shortlist rows `status: recommended`.
Signal 3/4/6/7 → all empty. Tier 1 (backlog) fires on the shortlist. Reply:

```text
1. "Duyệt 6 nhóm Facebook em vừa tìm được, để em theo dõi mỗi ngày." — needs: nothing, ready now.
2. "Chạy thêm một vòng quét nhóm riêng ngay bây giờ." — needs: nguồn riêng đã được duyệt (đã có).
3. "Thêm một khách hàng mới vào hệ thống." — needs: thông tin cơ bản của khách hàng đó.

Anh muốn bắt đầu với việc nào?
```
(offers: `promote_discovered_groups` tier 1, `scan_private_groups` tier 2, `add_client` tier 4 —
backlog first, one lead-gen option, one expansion option; the shortlist alone would already have
been enough to satisfy the gate, the other two round out the 2-3 slot.)

**B. Free install, 184 leads locked.** Poll: signal 1 → `entitlement.tier: free`. Signal 2 →
`locked: 184`, `unlocked: 30`, `max_contacts: 30`. This is `locked > 0` and the first time this
session — tier 5 fires and the reply carries the `review_locked_leads_upgrade` ACTION REQUIRED
block (not a next-jobs line), for example: "Em vừa đưa 214 lead mới vào CRM, 31 lead có tín hiệu
tốt. Gói Free đang mở 30 contact, 184 lead còn lại đang khoá chi tiết, chưa gửi mail hay nhắn tin
được. Mở gói Starter thì 500 contact mở ngay, không cần quét lại." Every reply after this one in
the same session drops the ACTION REQUIRED block and instead carries the persistent one-line meter
("184 leads locked under Free — 30/30 open") until the plan changes or the count changes.

**C. Paid install, idle.** Poll: signal 1 → `entitlement.tier: pro`. Signal 2 → `locked: 0`,
`unlocked: 340`/`max_contacts: 2000` (0.17, no approaching-cap line). Signals 3-7 all empty, no
lead landed this run. IDLE RULE applies — tier 2/4 only:

```text
1. "Chạy thêm một vòng Social Discovery Pass ngay bây giờ." — needs: nothing, ready now.
2. "Mở rộng từ khoá quét cho một thành phố hoặc ngành mới." — needs: tên thành phố/ngành đó.

Anh muốn em làm việc nào trước?
```

## Guardrails

- **Client-blind.** Nothing in this file — the CATALOGUE, the meter, the plan tier, `locked`
  counts, upgrade language — ever appears in the three client-facing HTML files, the combined
  client report/PDF, or the client notification. This file governs the Boss-facing Team Leader
  channel only.
- **Carve-outs stay carve-outs.** `entitlement.reason: seat_limit` and a stale-token
  `solo_feature_not_in_tier` on any capability that is not `write_actions` are never upsell
  moments — they mean "refresh the token" or "move the seat," never "offer an upgrade." Only
  `review_locked_leads_upgrade` and `write_actions_upgrade` are selling moments; every other row in
  the CATALOGUE is an offer of work, not a pitch.
- **Numbers come from state, never estimated.** Every `{N}`, `{L}`, `{unlocked}`, `{locked}` in an
  offer is read fresh from the STATE POLL source named in its row — never carried over from an
  earlier reply, never rounded, never invented because the real file was inconvenient to read.
- **No fixed-schedule nagging.** An offer fires because a signal is true right now, not because a
  fixed number of days has passed since it was last offered. `social_discovery_pass`, `make_video`,
  `review_analytics`, and similar recurring-feeling jobs are only offered when their actual signal
  (staleness measured from the real timestamp, or a concrete gap) is true this reply — repeating an
  offer the human already declined in this session, with no new signal, is nagging, not service.
- **Selling stays rare.** `review_locked_leads_upgrade` fires once per session at most (the first
  time `locked > 0` is seen); after that, the meter carries the number and no further ACTION
  REQUIRED upgrade block is shown unless the human asks. `write_actions_upgrade` fires only after a
  real refusal, never speculatively.
