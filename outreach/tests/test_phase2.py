#!/usr/bin/env python3
"""Phase 2 tests (stdlib unittest). Run: python3 -m unittest discover -s tests

2A — campaigns, segments, enrich-queue population with the don't-double-touch guards.
"""

import json
import os
import sys
import tempfile
import unittest

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(ROOT, "tools"))

import crm_store  # noqa: E402
from crm_store import CrmStore  # noqa: E402
from storage import now_iso  # noqa: E402


def _client():
    tmp = tempfile.mkdtemp()
    pipe = os.path.join(tmp, "daily-content-pipeline")
    crm_store.resolve_client_dir(pipe, "c", None, "b", "l", create=True)
    cdir = crm_store.resolve_client_dir(pipe, "c", None)
    CrmStore(cdir).init_tree()
    return cdir


def _write_sent(cdir, campaign, lead_id, sent_at):
    d = os.path.join(cdir, "campaigns", campaign, "sent", sent_at[:7])
    os.makedirs(d, exist_ok=True)
    with open(os.path.join(d, "sent_log.jsonl"), "a") as fh:
        fh.write(json.dumps({"seq": 1, "ts": sent_at, "lead_id": lead_id, "campaign": campaign,
                             "step": 1, "sendbox": "sb-a", "rfc_message_id": "<x@y>", "sent_at": sent_at}) + "\n")


class TestCampaignsSegmentsQueue(unittest.TestCase):
    def setUp(self):
        self.cdir = _client()
        self.s = CrmStore(self.cdir)
        # three leads with email, one without
        self.a, _ = self.s.add_contact({"name": {"full": "A"}, "identities": {"emails": [{"address": "a@x.com"}]}})
        self.b, _ = self.s.add_contact({"name": {"full": "B"}, "identities": {"emails": [{"address": "b@x.com"}]}})
        self.c, _ = self.s.add_contact({"name": {"full": "C"}, "identities": {"emails": [{"address": "c@x.com"}]}})
        self.noemail, _ = self.s.add_contact({"name": {"full": "NoEmail"}, "identities": {"phones": [{"number": "205-369-0520"}]}})
        self.s.set_segment({"id": "leads", "name": "All leads", "where": [["lifecycle_stage", "=", "lead"]]})
        self.s.create_campaign("demo", {"audience": {"segment": "leads"}, "channel_strategy": "email_first"})

    def test_goal_type_validated(self):
        with self.assertRaises(Exception):
            self.s.create_campaign("bad", {"goal": {"goal_type": "spam_blast"}})

    def test_segment_excludes_suppressed_and_merged(self):
        self.s.suppress("email", "a@x.com", "unsubscribe", tier="client")
        resolved = {c["id"] for c in self.s.resolve_segment("leads")}
        self.assertNotIn(self.a, resolved)          # suppressed excluded
        self.assertIn(self.b, resolved)

    def test_queue_email_first_queues_no_email_for_discovery(self):
        # email_first QUEUES a no-email lead so enrichment can DISCOVER an email — it is not an
        # up-front has-email gate. All four lifecycle=lead contacts are queued.
        r = self.s.queue_campaign("demo")
        self.assertEqual(r["queued"], 4)            # a, b, c, and the no-email lead
        self.assertEqual(r["skipped"]["no_email"], 0)

    def test_queue_no_email_skipped_when_recent_negative_cache(self):
        # a recent email_not_found_at means discovery already failed within the retry window -> skip
        self.s.set_contact(self.noemail, {"enrichment": {"email_not_found_at": now_iso()}})
        self.s.create_campaign("demo2", {"audience": {"segment": "leads"}, "channel_strategy": "email_first"})
        r = self.s.queue_campaign("demo2")
        self.assertEqual(r["queued"], 3)            # a, b, c
        self.assertEqual(r["skipped"]["no_email"], 1)

    def test_queue_idempotent(self):
        self.s.queue_campaign("demo")
        r2 = self.s.queue_campaign("demo")
        self.assertEqual(r2["queued"], 0)
        self.assertEqual(r2["skipped"]["already_in_campaign"], 4)

    def test_queue_skips_recently_touched_by_other_campaign(self):
        # A was emailed by campaign 'other' yesterday -> within the 7-day guard -> skip for 'demo'
        _write_sent(self.cdir, "other", self.a, _yesterday())
        r = self.s.queue_campaign("demo")
        self.assertEqual(r["skipped"]["recently_touched_elsewhere"], 1)
        self.assertEqual(r["queued"], 3)            # b, c, and the no-email lead

    def test_queue_skips_frozen_sequence(self):
        self.s.set_contact(self.b, {"sequence_state": "frozen"})  # replied, mid-handling
        r = self.s.queue_campaign("demo")
        self.assertEqual(r["skipped"]["in_active_sequence"], 1)
        self.assertEqual(r["queued"], 3)            # a, c, and the no-email lead


def _yesterday():
    import datetime as dt
    return (dt.datetime.now(dt.timezone.utc).replace(microsecond=0) - dt.timedelta(days=1)).isoformat().replace("+00:00", "Z")


class TestEnrich(unittest.TestCase):
    def setUp(self):
        self.cdir = _client()
        self.s = CrmStore(self.cdir)
        self.lead, _ = self.s.add_contact({"name": {"full": "Susan"}, "identities": {"emails": [{"address": "s@kw.com"}]}})
        self.s.create_campaign("demo", {"audience": {"segment": "x"}})

    def _dossier(self):
        return {"identity": {"still_active": "confirmed", "current_company": "KW",
                             "channels_found": {"emails": ["s.alt@kw.com"], "phones": []}},
                "context": {"market": "AL"},
                "hooks": [
                    {"type": "new_listing", "summary": "listed 123 Main St", "evidence_url": "https://z/1",
                     "observed_date": "2026-07-14", "confidence": 0.9, "analysis": {"sensitivity": "public_business"}},
                    {"type": "social_post", "summary": "no source", "confidence": 0.5, "analysis": {"sensitivity": "public_business"}},
                    {"type": "social_post", "summary": "kid birthday", "evidence_url": "https://fb/x", "analysis": {"sensitivity": "personal"}},
                ],
                "writing_brief": {"one_liner": "listing agent", "ranked_angles": ["new_listing"], "personalization_confidence": 0.85}}

    def test_hook_without_evidence_is_dropped(self):
        r = self.s.enrich_write(self.lead, self._dossier(), campaign_slug="demo")
        self.assertEqual(r["usable_hooks"], 1)      # only the evidenced public hook
        self.assertTrue(any("evidence_url" in p for p in r["problems"]))

    def test_personal_hook_barred_to_do_not_mention(self):
        r = self.s.enrich_write(self.lead, self._dossier(), campaign_slug="demo")
        self.assertGreaterEqual(r["do_not_mention"], 1)
        en = self.s.get_contact(self.lead)["enrichment"]
        self.assertNotIn("kid birthday", [h["summary"] for h in en["hooks"]])  # never a usable hook
        self.assertIn("kid birthday", en["writing_brief"]["do_not_mention"])

    def test_confidence_band(self):
        r = self.s.enrich_write(self.lead, self._dossier(), campaign_slug="demo")
        self.assertEqual(r["confidence_band"], "high")

    def test_found_email_stored_as_enrich_source(self):
        self.s.enrich_write(self.lead, self._dossier(), campaign_slug="demo")
        emails = {e["address"]: e for e in self.s.get_contact(self.lead)["identities"]["emails"]}
        self.assertIn("s.alt@kw.com", emails)
        self.assertEqual(emails["s.alt@kw.com"]["source"], "enrich")  # found, never 'guess'

    def test_status_ttl_and_inheritance(self):
        self.assertEqual(self.s.enrich_status(self.lead)["needs"], "enrich")
        self.s.enrich_write(self.lead, self._dossier(), campaign_slug="demo")
        self.assertEqual(self.s.enrich_status(self.lead)["needs"], "skip")   # fresh -> reused (inheritance)
        # hooks age out after HOOK_TTL_DAYS while identity stays fresh -> refresh, not full re-enrich
        future = crm_store._iso_days_ago(-(CrmStore.HOOK_TTL_DAYS + 1))       # now + (TTL+1) days
        self.assertEqual(self.s.enrich_status(self.lead, now=future)["needs"], "refresh")

    def test_enrich_due_lists_only_stale(self):
        self.s.set_segment({"id": "all", "name": "all", "where": [["lifecycle_stage", "=", "lead"]]})
        self.s.create_campaign("c2", {"audience": {"segment": "all"}})
        self.s.queue_campaign("c2")
        due_before = self.s.enrich_due("c2")
        self.assertTrue(any(d["lead_id"] == self.lead for d in due_before))
        self.s.enrich_write(self.lead, self._dossier(), campaign_slug="c2")
        due_after = self.s.enrich_due("c2")
        self.assertFalse(any(d["lead_id"] == self.lead for d in due_after))  # now fresh -> not due


class TestDraftWriting(unittest.TestCase):
    def setUp(self):
        self.cdir = _client()
        self.s = CrmStore(self.cdir)
        import gmail_client as g
        self.lead, _ = self.s.add_contact({"name": {"full": "Susan"}, "identities": {"emails": [{"address": "s@kw.com", "is_primary": True}]}})
        g.save_sendbox(self.cdir, {"slug": "sb-a", "email": "me@gmail.com", "domain": "gmail.com", "quota_today": 40, "status": "healthy", "imap_uid_cursor": 0})
        g.save_sendbox(self.cdir, {"slug": "sb-b", "email": "me2@gmail.com", "domain": "gmail.com", "quota_today": 40, "status": "needs_reauth", "imap_uid_cursor": 0})
        # this campaign OPTS IN to a generic opener so the hookless step-1 cases below are allowed
        self.s.create_campaign("demo", {"audience": {"segment": "x", "personalization": {"no_hook_fallback": "generic_honest_opener"}}, "sendboxes": ["sb-a", "sb-b"]})
        self.s.enrich_write(self.lead, {"identity": {"still_active": "confirmed"},
                            "hooks": [{"type": "new_listing", "summary": "listed 123 Main St", "evidence_url": "https://z/1",
                                       "observed_date": "2026-07-14", "confidence": 0.9, "analysis": {"sensitivity": "public_business"}}],
                            "writing_brief": {"personalization_confidence": 0.85}}, campaign_slug="demo")

    def test_draft_ok_and_rotation_skips_unhealthy(self):
        r = self.s.draft_write(self.lead, "demo", 1, "Idea for your listing", "Hi...", hooks_used=[{"type": "new_listing", "evidence_url": "https://z/1"}])
        self.assertEqual(r["sendbox"], "sb-a")     # sb-b is needs_reauth
        self.assertEqual(r["confidence_band"], "high")
        self.assertEqual(r["warnings"], [])

    def test_draft_rejects_unsourced_hook(self):
        with self.assertRaises(Exception):
            self.s.draft_write(self.lead, "demo", 1, "x", "y", hooks_used=[{"type": "new_listing", "evidence_url": "https://FAKE"}])

    def test_draft_rejects_step1_fake_reply(self):
        with self.assertRaises(Exception):
            self.s.draft_write(self.lead, "demo", 1, "Re: following up", "y")

    def test_hook_marked_used_in(self):
        self.s.draft_write(self.lead, "demo", 1, "Idea", "Hi", hooks_used=[{"type": "new_listing", "evidence_url": "https://z/1"}])
        self.assertIn("demo/step1", self.s.get_contact(self.lead)["enrichment"]["hooks"][0]["used_in"])

    def test_generic_opener_warning_when_no_hook(self):
        r = self.s.draft_write(self.lead, "demo", 1, "Hello", "Hi, generic opener")
        self.assertIn("generic_opener", r["warnings"])

    def test_step1_hookless_rejected_on_default_campaign(self):
        # a default campaign (no_hook_fallback=skip) rejects a step-1 email with no evidenced hook
        self.s.create_campaign("proof", {"audience": {"segment": "x"}, "sendboxes": ["sb-a"]})
        with self.assertRaises(Exception) as ctx:
            self.s.draft_write(self.lead, "proof", 1, "Hello", "no hook body")
        self.assertIn("no_evidenced_hook", str(ctx.exception))

    def test_step1_hookless_allowed_with_generic_optin(self):
        # the SAME hookless draft is accepted (with the generic_opener flag) when the campaign opts in
        self.s.create_campaign("proof_generic", {"audience": {"segment": "x", "personalization": {"no_hook_fallback": "generic_honest_opener"}}, "sendboxes": ["sb-a"]})
        r = self.s.draft_write(self.lead, "proof_generic", 1, "Hello", "generic opener body")
        self.assertIn("generic_opener", r["warnings"])

    def test_draft_budget_remaining_math(self):
        self.s.create_campaign("budgeted", {"audience": {"segment": "x", "personalization": {"no_hook_fallback": "generic_honest_opener"}}, "sendboxes": ["sb-a"], "daily_quota": 2})
        b0 = self.s.draft_budget("budgeted")
        self.assertEqual(b0["daily_quota"], 2)
        self.assertEqual(b0["remaining"], 2)
        self.s.draft_write(self.lead, "budgeted", 1, "Hello", "generic")   # one draft created today
        b1 = self.s.draft_budget("budgeted")
        self.assertEqual(b1["used_today"], 1)
        self.assertEqual(b1["remaining"], 1)

    def test_sticky_sender_on_draft(self):
        self.s.set_contact(self.lead, {"assigned_sendbox": "sb-b"})  # already assigned (even if 'unhealthy')
        r = self.s.draft_write(self.lead, "demo", 2, "Re: Idea", "bump", hooks_used=[{"type": "new_listing", "evidence_url": "https://z/1"}])
        self.assertEqual(r["sendbox"], "sb-b")     # sticky sender wins over rotation for a bump
        self.assertIn("bump_step", r["warnings"])


class TestApprovalAndFollowup(unittest.TestCase):
    def setUp(self):
        self.cdir = _client()
        self.s = CrmStore(self.cdir)
        import gmail_client as g
        g.save_sendbox(self.cdir, {"slug": "sb-a", "email": "me@gmail.com", "domain": "gmail.com", "quota_today": 40, "status": "healthy", "imap_uid_cursor": 0})
        # opt in to a generic opener so the third (hookless) step-1 lead below can be drafted
        self.s.create_campaign("demo", {"audience": {"segment": "x", "personalization": {"no_hook_fallback": "generic_honest_opener"}}, "sendboxes": ["sb-a"]})
        self.leads = []
        for i, hook in enumerate([True, True, False]):
            lead, _ = self.s.add_contact({"name": {"full": f"L{i}"}, "identities": {"emails": [{"address": f"l{i}@x.com", "is_primary": True}]}})
            self.leads.append(lead)
            if hook:
                self.s.enrich_write(lead, {"identity": {"still_active": "confirmed"}, "hooks": [{"type": "new_listing", "summary": "x", "evidence_url": f"https://z/{i}", "observed_date": "2026-07-14", "confidence": 0.9, "analysis": {"sensitivity": "public_business"}}], "writing_brief": {"personalization_confidence": 0.85}}, campaign_slug="demo")
                self.s.draft_write(lead, "demo", 1, f"Idea {i}", "Hi", hooks_used=[{"type": "new_listing", "evidence_url": f"https://z/{i}"}])
            else:
                self.s.enrich_write(lead, {"identity": {"still_active": "confirmed"}, "hooks": [], "writing_brief": {"personalization_confidence": 0.3}}, campaign_slug="demo")
                self.s.draft_write(lead, "demo", 1, f"Hello {i}", "generic")

    def test_approval_report_groups_and_numbers(self):
        md, index = self.s.build_approval()
        self.assertEqual(len(index), 3)
        self.assertIn("High confidence (2)", md)
        self.assertIn("Review carefully (1)", md)

    def test_approval_report_followups_section(self):
        # a step-2 bump lands in the dedicated Follow-ups section; step-1 drafts stay in High/Review
        self.s.draft_write(self.leads[0], "demo", 2, "Re: Idea 0", "just following up",
                           hooks_used=[{"type": "new_listing", "evidence_url": "https://z/0"}])
        md, index = self.s.build_approval()
        self.assertEqual(len(index), 4)                     # 3 step-1 + 1 step-2
        self.assertIn("High confidence (2)", md)            # step-1 grouping unchanged
        self.assertIn("Review carefully (1)", md)
        self.assertIn("Follow-ups due (1)", md)

    def test_approve_moves_to_approved_and_logs(self):
        self.s.render_approval_report()
        res = self.s.approve_apply({"approve": "1,2", "reject": [{"n": 3, "reason": "too generic"}]})
        self.assertEqual(len(res["approved"]), 2)
        self.assertEqual(len(res["rejected"]), 1)
        self.assertEqual(len(self.s.list_pending_drafts()), 0)          # all decided
        approved = [f for f in os.listdir(os.path.join(self.cdir, "campaigns", "demo", "outbox", "approved")) if f.endswith(".json")]
        self.assertEqual(len(approved), 2)
        with open(os.path.join(self.cdir, "campaigns", "demo", "outbox", "approved", approved[0])) as fh:
            d = json.load(fh)
        self.assertEqual(d["status"], "approved")                        # send engine can now send it
        with open(os.path.join(self.cdir, "analytics", "learning_log.md")) as fh:
            self.assertIn("too generic", fh.read())

    def test_approve_all(self):
        self.s.render_approval_report()
        res = self.s.approve_apply({"approve": "all"})
        self.assertEqual(len(res["approved"]), 3)

    def test_edit_then_approve_uses_new_body(self):
        self.s.render_approval_report()
        self.s.approve_apply({"edit": [{"n": 1, "body_text": "edited body"}], "approve": "1"})
        approved = [f for f in os.listdir(os.path.join(self.cdir, "campaigns", "demo", "outbox", "approved")) if f.endswith(".json")]
        bodies = []
        for f in approved:
            with open(os.path.join(self.cdir, "campaigns", "demo", "outbox", "approved", f)) as fh:
                bodies.append(json.load(fh)["body_text"])
        self.assertIn("edited body", bodies)

    def test_followups_due_and_reply_freeze(self):
        import datetime as dt
        lead = self.leads[0]
        sa = (dt.datetime.now(dt.timezone.utc).replace(microsecond=0) - dt.timedelta(days=5)).isoformat().replace("+00:00", "Z")
        d = os.path.join(self.cdir, "campaigns", "demo", "sent", sa[:7]); os.makedirs(d, exist_ok=True)
        with open(os.path.join(d, "sent_log.jsonl"), "a") as fh:
            fh.write(json.dumps({"seq": 1, "ts": sa, "lead_id": lead, "campaign": "demo", "step": 1, "sendbox": "sb-a", "rfc_message_id": "<x>", "sent_at": sa}) + "\n")
        due = self.s.followups_due("demo")
        self.assertEqual([x["lead_id"] for x in due], [lead])           # step-1 5d ago, gap 4 elapsed
        self.assertEqual(due[0]["next_step"], 2)
        self.s.set_contact(lead, {"sequence_state": "frozen"})           # they replied
        self.assertEqual(self.s.followups_due("demo"), [])              # a reply freezes the bump

    def test_today_view_and_kanban_render(self):
        self.assertTrue(self.s.render_today_view()["html_rendered"])
        self.assertTrue(self.s.render_kanban()["html_rendered"])


class TestUIIngest(unittest.TestCase):
    """ui_inbox/approval_decisions.jsonl (bridge UI) -> same terminal semantics as chat approvals."""

    def setUp(self):
        self.cdir = _client()
        self.s = CrmStore(self.cdir)
        import gmail_client as g
        g.save_sendbox(self.cdir, {"slug": "sb-a", "email": "me@gmail.com", "domain": "gmail.com", "quota_today": 40, "status": "healthy", "imap_uid_cursor": 0})
        self.s.create_campaign("demo", {"audience": {"segment": "x"}, "sendboxes": ["sb-a"]})
        self.drafts = []
        for i in range(3):
            lead, _ = self.s.add_contact({"name": {"full": f"U{i}"}, "identities": {"emails": [{"address": f"u{i}@x.com", "is_primary": True}]}})
            self.s.enrich_write(lead, {"identity": {"still_active": "confirmed"}, "hooks": [{"type": "new_listing", "summary": "x", "evidence_url": f"https://u/{i}", "observed_date": "2026-07-14", "confidence": 0.9, "analysis": {"sensitivity": "public_business"}}], "writing_brief": {"personalization_confidence": 0.85}}, campaign_slug="demo")
            r = self.s.draft_write(lead, "demo", 1, f"Idea {i}", "Hi", hooks_used=[{"type": "new_listing", "evidence_url": f"https://u/{i}"}])
            self.drafts.append(r["draft_id"])

    def _write_inbox(self, decisions):
        d = os.path.join(self.cdir, "ui_inbox")
        os.makedirs(d, exist_ok=True)
        with open(os.path.join(d, "approval_decisions.jsonl"), "a", encoding="utf-8") as fh:
            for dec in decisions:
                fh.write(json.dumps(dec) + "\n")

    def test_ingest_applies_terminal_decisions_and_is_idempotent(self):
        self._write_inbox([
            {"ts": "t", "draft_id": self.drafts[0], "campaign": "demo", "decision": "approve",
             "edited_body": "ui edited body", "ui_session": "s1"},
            {"ts": "t", "draft_id": self.drafts[1], "campaign": "demo", "decision": "reject",
             "note": "wrong angle", "ui_session": "s1"},
            {"ts": "t", "draft_id": self.drafts[2], "decision": "hold", "ui_session": "s1"},
            {"ts": "t", "draft_id": "draft_MISSING", "campaign": "demo", "decision": "approve", "ui_session": "s1"},
        ])
        res = self.s.ingest_ui_decisions()
        self.assertEqual(len(res["approved"]), 1)
        self.assertEqual(res["rejected"], [self.drafts[1]])
        self.assertEqual(res["held"], [self.drafts[2]])
        self.assertIn("draft_MISSING", res["not_found"])
        # approved draft moved with the UI edit applied
        dest = os.path.join(self.cdir, "campaigns", "demo", "outbox", "approved", f"{self.drafts[0]}.json")
        with open(dest) as fh:
            d = json.load(fh)
        self.assertEqual(d["status"], "approved")
        self.assertEqual(d["decided_by"], "ui")
        self.assertEqual(d["body_text"], "ui edited body")
        # ledger + learning log written
        with open(os.path.join(self.cdir, "approvals", "approval_log.md")) as fh:
            log = fh.read()
        self.assertIn("| ui |", log)
        with open(os.path.join(self.cdir, "analytics", "learning_log.md")) as fh:
            self.assertIn("wrong angle", fh.read())
        # idempotent: second ingest processes nothing new
        res2 = self.s.ingest_ui_decisions()
        self.assertEqual(res2["approved"], [])
        self.assertEqual(res2["rejected"], [])
        # new decision after cursor still ingests (hold released to approve is NOT allowed:
        # the held draft is no longer pending_approval, so it reports already_processed)
        self._write_inbox([{"ts": "t2", "draft_id": self.drafts[2], "campaign": "demo", "decision": "approve", "ui_session": "s2"}])
        res3 = self.s.ingest_ui_decisions()
        self.assertEqual(res3["already_processed"], [self.drafts[2]])

    def test_ingest_no_inbox_is_noop(self):
        res = self.s.ingest_ui_decisions()
        self.assertEqual(res["processed_lines"], 0)


class TestInjectableClock(unittest.TestCase):
    """2E — OUTREACHCRM_FAKE_NOW, gated behind OUTREACHCRM_TEST_MODE (DESIGN §17)."""

    def setUp(self):
        self._saved = {k: os.environ.get(k) for k in ("OUTREACHCRM_TEST_MODE", "OUTREACHCRM_FAKE_NOW")}

    def tearDown(self):
        for k, v in self._saved.items():
            if v is None:
                os.environ.pop(k, None)
            else:
                os.environ[k] = v

    def test_fake_now_ignored_without_test_mode(self):
        os.environ.pop("OUTREACHCRM_TEST_MODE", None)
        os.environ["OUTREACHCRM_FAKE_NOW"] = "2000-01-01T00:00:00Z"
        self.assertFalse(now_iso().startswith("2000"))     # production is never shifted

    def test_fake_now_applies_under_test_mode(self):
        os.environ["OUTREACHCRM_TEST_MODE"] = "1"
        os.environ["OUTREACHCRM_FAKE_NOW"] = "2026-08-15T09:00:00Z"
        from storage import today_str, month_str
        self.assertEqual(now_iso(), "2026-08-15T09:00:00Z")
        self.assertEqual(today_str(), "2026-08-15")         # derives from now_iso
        self.assertEqual(month_str(), "2026-08")
        os.environ["OUTREACHCRM_FAKE_NOW"] = "2026-08-15"   # date-only -> midnight
        self.assertEqual(now_iso(), "2026-08-15T00:00:00Z")

    def test_malformed_fake_now_raises_only_in_test_mode(self):
        from storage import StorageError
        os.environ["OUTREACHCRM_TEST_MODE"] = "1"
        os.environ["OUTREACHCRM_FAKE_NOW"] = "not-a-date"
        with self.assertRaises(StorageError):
            now_iso()
        os.environ.pop("OUTREACHCRM_TEST_MODE", None)        # inert (real now) outside test mode
        self.assertFalse(now_iso().startswith("not"))

    def test_time_advance_makes_bump_due(self):
        os.environ["OUTREACHCRM_TEST_MODE"] = "1"
        cdir = _client(); s = CrmStore(cdir)
        s.create_campaign("demo", {"audience": {"segment": "x"},
                                   "sequence": [{"step": 1, "gap_days": 0}, {"step": 2, "gap_days": 4}]})
        lead, _ = s.add_contact({"identities": {"emails": [{"address": "a@x.com"}]}})
        os.environ["OUTREACHCRM_FAKE_NOW"] = "2026-07-10T09:00:00Z"
        _write_sent(cdir, "demo", lead, now_iso())
        self.assertEqual(s.followups_due("demo"), [])        # gap not elapsed at day 0
        os.environ["OUTREACHCRM_FAKE_NOW"] = "2026-07-15T09:00:00Z"
        self.assertEqual(len(s.followups_due("demo")), 1)    # gap elapsed after +5d


class TestWeeklyReport(unittest.TestCase):
    """2E — minimal client-facing weekly report (scrub-gated)."""

    def setUp(self):
        os.environ["OUTREACHCRM_TEST_MODE"] = "1"
        os.environ["OUTREACHCRM_FAKE_NOW"] = "2026-07-16T10:00:00Z"
        self.cdir = _client(); self.s = CrmStore(self.cdir)
        self.s.create_campaign("demo", {"audience": {"segment": "x"}})

    def tearDown(self):
        for k in ("OUTREACHCRM_TEST_MODE", "OUTREACHCRM_FAKE_NOW"):
            os.environ.pop(k, None)

    def _seed(self, name="Jane Smith"):
        lead, _ = self.s.add_contact({"name": {"full": name}, "identities": {"emails": [{"address": f"{name.split()[0].lower()}@x.com"}]}})
        for _ in range(10):
            self.s.log_activity("email_sent", lead, "sent")
        self.s.log_activity("email_reply", lead, "reply")
        self.s.apply_rules([{"type": "reply_positive", "contact_id": lead, "activity_id": f"a-{name}"}])
        return lead

    def test_clean_render_is_client_facing(self):
        self._seed()
        r = self.s.render_weekly_report(client_name="Acme Inc")
        self.assertFalse(r["blocked"])
        self.assertTrue(r["html_rendered"])
        with open(r["md"]) as fh:
            md = fh.read()
        self.assertIn("Acme Inc — Weekly Outreach Report", md)
        self.assertIn("Emails delivered", md)
        for term in ("sendbox", "crm_store", "WideCast", "OutreachCRM", "campaign"):
            self.assertNotIn(term, md)                       # source is built clean

    def test_scrub_gate_blocks_blind_term(self):
        self._seed("WideCast Holdings")                       # a blind term reaches a movement line
        r = self.s.render_weekly_report(client_name="Acme Inc")
        self.assertTrue(r["blocked"])
        self.assertIn("WideCast", r["blind_terms"])
        self.assertIsNone(r["html"])                          # contaminated report is never shipped

    def test_data_counts(self):
        self._seed()
        d = self.s.weekly_report_data()
        self.assertEqual(d["delivered"], 10)
        self.assertEqual(d["replies"], 1)
        self.assertEqual(d["new_conversations"], 1)
        self.assertEqual(d["period_start"], "2026-07-09")

    def test_monthly_report_renders_and_windows(self):
        self._seed()                                          # July (current-month) activity
        r = self.s.render_monthly_report(client_name="Acme Inc")
        self.assertFalse(r["blocked"])
        self.assertTrue(r["html_rendered"])
        with open(r["md"]) as fh:
            self.assertIn("Monthly Outreach Report", fh.read())
        # a prior-month window excludes this month's activity
        june = self.s.monthly_report_data(month="2026-06")
        self.assertEqual(june["delivered"], 0)
        self.assertEqual(june["period_start"], "2026-06-01")
        self.assertEqual(june["period_end"], "2026-07-01")   # end-exclusive = first day of next month
        # the current month is month-to-date (window ends at now)
        july = self.s.monthly_report_data()
        self.assertEqual(july["delivered"], 10)
        self.assertEqual(july["period_start"], "2026-07-01")


class TestNotify(unittest.TestCase):
    """2E — composed operator notification (offline/degraded paths)."""

    def setUp(self):
        sys.path.insert(0, os.path.join(ROOT, "tools"))
        import provider_openapi  # noqa: E402
        self.po = provider_openapi
        self.tmp = tempfile.mkdtemp()

    def _run(self, argv):
        import io, contextlib
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            rc = self.po.main(argv)
        return rc, json.loads(buf.getvalue())

    def _cfg(self, block):
        p = os.path.join(self.tmp, f"cfg_{len(os.listdir(self.tmp))}.json")
        with open(p, "w") as fh:
            json.dump({"active_provider": "widecast", "providers": {"widecast": block}}, fh)
        return p

    def test_missing_config_is_local_path_only(self):
        rc, j = self._run(["notify", "--message", "done"])
        self.assertEqual(rc, 0)
        self.assertEqual(j["status"], "local_path_only")
        self.assertEqual(j["blocker"], "provider_config_missing")

    def test_disabled_notification_is_degraded_not_fatal(self):
        cfg = self._cfg({"notification": {"enabled": False}, "api_key_local": "wc_live_x"})
        rc, j = self._run(["--config", cfg, "notify", "--message", "done"])
        self.assertEqual(rc, 0)
        self.assertEqual(j["blocker"], "provider_notification_not_configured")

    def test_enabled_without_key_is_auth_missing(self):
        cfg = self._cfg({"notification": {"enabled": True}, "api_key_env": "UNSET_X", "api_key_local": ""})
        rc, j = self._run(["--config", cfg, "notify", "--message", "done"])
        self.assertEqual(j["blocker"], "provider_auth_missing")

    def test_dry_run_writes_log_and_no_send(self):
        cfg = self._cfg({"notification": {"enabled": True}, "api_key_local": "wc_live_x"})
        log = os.path.join(self.tmp, "notifications", "notification_log.md")
        rc, j = self._run(["--config", cfg, "notify", "--message", "done", "--dry-run",
                           "--event", "weekly_client_report_ready", "--log", log])
        self.assertEqual(rc, 0)
        self.assertEqual(j["status"], "dry_run")
        self.assertTrue(j["dry_run"])
        with open(log) as fh:
            body = fh.read()
        self.assertIn("# Notification Log", body)
        self.assertIn("weekly_client_report_ready", body)
        self.assertIn("dry_run", body)


class TestAuditFixes(unittest.TestCase):
    """Regressions for the adversarial-audit findings (32 confirmed). One assertion per fix."""

    def setUp(self):
        self.cdir = _client(); self.s = CrmStore(self.cdir)
        import gmail_client as g
        self.g = g

    def _sb(self, slug="sb-a", quota=40, status="healthy"):
        self.g.save_sendbox(self.cdir, {"slug": slug, "email": f"{slug}@gmail.com", "domain": "gmail.com",
                                        "quota_today": quota, "status": status, "imap_uid_cursor": 0})

    def _deal(self, extra):
        c, _ = self.s.add_contact({"name": {"full": "P"}, "identities": {"emails": [{"address": "p@x.com"}]}})
        return self.s.create_deal(c, "new_reply", extra=extra)

    # --- blocker / crashes ---
    def test_null_value_deal_does_not_crash_kanban_or_weekly(self):
        self._deal({"value": None})
        self.assertTrue(self.s.render_kanban()["html_rendered"])       # no TypeError
        self.assertIsInstance(self.s.weekly_report_data()["forecast"], float)

    def test_enrich_non_numeric_confidence_does_not_crash(self):
        c, _ = self.s.add_contact({"identities": {"emails": [{"address": "a@x.com"}]}})
        r = self.s.enrich_write(c, {"hooks": [{"type": "x", "evidence_url": "https://z/1", "confidence": "high",
                                               "analysis": {"sensitivity": "public_business"}}],
                                    "writing_brief": {"personalization_confidence": "very"}})
        self.assertEqual(r["usable_hooks"], 1)                          # coerced, not crashed

    # --- approval gate ---
    def test_approve_is_idempotent_on_reapply(self):
        self._prep_draft(); self.s.render_approval_report()
        self.s.approve_apply({"approve": "1"})
        r2 = self.s.approve_apply({"approve": "1"})                     # must not raise FileNotFoundError
        self.assertEqual(r2["approved"], [])
        self.assertIn("draft", r2["already_processed"][0] if r2["already_processed"] else "")

    def test_approve_atomic_move_no_duplicate_pending(self):
        self._prep_draft(); self.s.render_approval_report()
        self.s.approve_apply({"approve": "1"})
        pend = os.path.join(self.cdir, "campaigns", "demo", "outbox", "pending_approval")
        left = [f for _, _, fs in os.walk(pend) for f in fs if f.endswith(".json")]
        self.assertEqual(left, [])                                      # no stale pending copy

    def test_approval_numbering_stable_across_rerender(self):
        self._prep_draft(n=3); self.s.render_approval_report()
        with open(self._idx()) as fh:
            idx1 = json.load(fh)["index"]
        first_id = next(e["draft_id"] for e in idx1 if e["n"] == 1)
        self.s.approve_apply({"approve": "1"})                          # remove #1
        self.s.render_approval_report()                                 # re-render
        with open(self._idx()) as fh:
            idx2 = {e["n"]: e["draft_id"] for e in json.load(fh)["index"]}
        self.assertNotIn(first_id, idx2.values())
        self.assertNotIn(1, idx2)                                       # #1 retired, not reused
        for n, did in idx2.items():
            self.assertEqual(n, next(e["n"] for e in idx1 if e["draft_id"] == did))  # numbers held

    def test_resolve_numbers_reversed_and_bounded(self):
        self._prep_draft(n=3); self.s.render_approval_report()
        self.assertEqual(len(self.s._resolve_numbers("3-1")), 3)        # reversed tolerated
        with self.assertRaises(Exception):
            self.s._resolve_numbers("1-99999999")                      # huge span rejected

    def test_approve_reports_not_found(self):
        self._prep_draft(); self.s.render_approval_report()
        r = self.s.approve_apply({"approve": [1, 99]})
        self.assertIn(99, r["not_found"])

    def test_reject_then_hold_same_number_consistent(self):
        self._prep_draft(); self.s.render_approval_report()
        r = self.s.approve_apply({"reject": [{"n": 1, "reason": "x"}], "hold": [1]})
        self.assertTrue(r["rejected"]); self.assertFalse(r["held"])  # hold on same n skipped, reject won
        with open(self.s._resolve_numbers([1])[0]["path"]) as fh:
            d = json.load(fh)
        self.assertEqual(d["status"], "rejected")

    # --- safety gates ---
    def test_step1_re_bypass_unicode_blocked(self):
        self._prep_draft(write=False)
        for subj in ["​Re: hi", "Re： hi", "Ｒｅ: hi"]:
            with self.assertRaises(Exception):
                self.s.draft_write(self.lead, "demo", 1, subj, "b",
                                   hooks_used=[{"type": "t", "evidence_url": "https://z/1"}])

    def test_campaign_slug_traversal_rejected(self):
        with self.assertRaises(Exception):
            self.s.create_campaign("../../evil", {"audience": {"segment": "x"}})

    def test_evidence_url_placeholder_rejected(self):
        c, _ = self.s.add_contact({"identities": {"emails": [{"address": "a@x.com"}]}})
        r = self.s.enrich_write(c, {"hooks": [{"type": "x", "evidence_url": "N/A",
                                               "analysis": {"sensitivity": "public_business"}}],
                                    "writing_brief": {"personalization_confidence": 0.9}})
        self.assertEqual(r["usable_hooks"], 0)

    def test_draft_uses_dossier_hook_type_not_caller(self):
        self._prep_draft(write=False)
        r = self.s.draft_write(self.lead, "demo", 1, "Idea", "b",
                               hooks_used=[{"type": "FABRICATED_AWARD", "evidence_url": "https://z/1"}])
        with open(r["path"]) as fh:
            d = json.load(fh)
        self.assertEqual(d["hooks_used"][0]["type"], "new_listing")     # dossier type wins

    def test_email_first_queues_garbage_email_but_draft_still_rejects(self):
        # a garbage-email contact is QUEUED for email discovery (queued=1), but the email
        # requirement still hard-gates at draft time — draft_write to it raises (no usable email).
        self.s.create_campaign("c2", {"audience": {"segment": "seg"}, "channel_strategy": "email_first"})
        bad, _ = self.s.add_contact({"name": {"full": "Bad"}, "identities": {"emails": [{"address": "not-an-email"}]}})
        self.s.set_segment({"id": "seg", "where": [["lifecycle_stage", "=", "lead"]]})
        r = self.s.queue_campaign("c2")
        self.assertEqual(r["queued"], 1)
        self.assertEqual(r["skipped"]["no_email"], 0)
        with self.assertRaises(Exception):
            self.s.draft_write(bad, "c2", 1, "Hi", "body")

    # --- merge integrity ---
    def test_queue_does_not_requeue_merged_winner(self):
        self.s.create_campaign("demo", {"audience": {"segment": "seg"}})
        loser, _ = self.s.add_contact({"identities": {"emails": [{"address": "l@x.com"}]}})
        winner, _ = self.s.add_contact({"identities": {"emails": [{"address": "w@x.com"}]}})
        _write_sent(self.cdir, "demo", loser, now_iso())               # loser was emailed
        self.s.merge(loser, winner)
        self.s.set_segment({"id": "seg", "where": [["lifecycle_stage", "=", "lead"]]})
        r = self.s.queue_campaign("demo")
        with open(os.path.join(self.cdir, "campaigns", "demo", "queue", "enrich_queue.jsonl")) as fh:
            rows = [json.loads(l) for l in fh]
        self.assertNotIn(winner, [x["lead_id"] for x in rows])         # winner already touched via loser

    def test_followups_due_dedupes_merged_ids(self):
        self.s.create_campaign("demo", {"audience": {"segment": "x"},
                                        "sequence": [{"step": 1, "gap_days": 0}, {"step": 2, "gap_days": 0}]})
        loser, _ = self.s.add_contact({"identities": {"emails": [{"address": "l@x.com"}]}})
        winner, _ = self.s.add_contact({"identities": {"emails": [{"address": "w@x.com"}]}})
        _write_sent(self.cdir, "demo", loser, now_iso())
        _write_sent(self.cdir, "demo", winner, now_iso())
        self.s.merge(loser, winner)
        due = self.s.followups_due("demo")
        self.assertEqual(len({d["lead_id"] for d in due}), 1)          # one real contact, one row

    def test_apply_rules_no_duplicate_task_across_merged_ids(self):
        loser, _ = self.s.add_contact({"identities": {"emails": [{"address": "l@x.com"}]}})
        winner, _ = self.s.add_contact({"identities": {"emails": [{"address": "w@x.com"}]}})
        self.s.apply_rules([{"type": "reply_positive", "contact_id": loser, "activity_id": "a1"}])
        self.s.merge(loser, winner)
        self.s.apply_rules([{"type": "reply_positive", "contact_id": winner, "activity_id": "a2"}])
        open_tasks = [t for t in self.s._latest_tasks() if t.get("status") == "open"
                      and t.get("contact_id") == self.s.resolve(winner)]
        self.assertEqual(len(open_tasks), 1)                            # not duplicated

    # --- enrich integrity ---
    def test_hooks_only_refresh_preserves_identity(self):
        c, _ = self.s.add_contact({"identities": {"emails": [{"address": "a@x.com"}]}})
        self.s.enrich_write(c, {"identity": {"still_active": "inactive"},
                                "hooks": [], "writing_brief": {"personalization_confidence": 0.5}})
        self.s.enrich_write(c, {"hooks": [{"type": "x", "evidence_url": "https://z/1",
                                           "analysis": {"sensitivity": "public_business"}}],
                                "writing_brief": {"personalization_confidence": 0.8}})  # no identity key
        ident = self.s.get_contact(c)["enrichment"]["identity"]
        self.assertEqual(ident["still_active"], "inactive")            # not wiped

    def test_neg_cache_email_not_found_recent_is_skipped(self):
        import datetime as dt
        c, _ = self.s.add_contact({"identities": {"phones": [{"number": "+15551230000"}]}})
        recent = (dt.datetime.now(dt.timezone.utc) - dt.timedelta(days=11)).replace(microsecond=0).isoformat().replace("+00:00", "Z")
        self.s.set_contact(c, {"enrichment": {"identity": {"still_active": "confirmed", "enriched_at": now_iso()},
                                              "hooks_refreshed_at": now_iso(), "email_not_found_at": recent}})
        self.assertEqual(self.s.enrich_status(c)["reason"], "email_not_found_recent")

    # --- segment DSL / sendbox ---
    def test_cond_in_string_value_is_not_substring(self):
        c, _ = self.s.add_contact({"identities": {"emails": [{"address": "a@x.com"}]}})
        self.s.set_segment({"id": "seg", "where": [["lifecycle_stage", "in", "leadership"]]})
        self.assertEqual(self.s.resolve_segment("seg"), [])            # 'lead' not substring-matched

    def test_pick_sendbox_excludes_exhausted_box(self):
        self._sb("sb-full", quota=0); self._sb("sb-ok", quota=40)
        c, _ = self.s.add_contact({"identities": {"emails": [{"address": "a@x.com"}]}})
        box = self.s.pick_sendbox({"sendboxes": ["sb-full", "sb-ok"]}, self.s.get_contact(c))
        self.assertEqual(box, "sb-ok")

    # --- today view / weekly ---
    def test_today_view_flags_deal_missing_timestamps(self):
        d = self._deal({"value": 100})
        # strip stage_history + created_at to simulate a migrated record
        self.s.a.update("deals", d["id"], lambda x: {**x, "stage_history": [], "created_at": ""})
        self.assertTrue(any(b["deal_id"] == d["id"] for b in self.s.today_view_data()["sla_breaches"]))

    def test_reply_rate_capped_at_100(self):
        c, _ = self.s.add_contact({"identities": {"emails": [{"address": "a@x.com"}]}})
        self.s.log_activity("email_sent", c, "s")
        for _ in range(3):
            self.s.log_activity("email_reply", c, "r")
        self.assertLessEqual(self.s.weekly_report_data()["reply_rate"], 1.0)

    def test_scrub_gate_catches_term_split_across_markup(self):
        import report_renderer as rr
        # 'API key' broken across a <strong> tag must still be caught
        html = "<p>API <strong>key</strong> Ventures reached out</p>"
        self.assertIn("API key", rr.scrub_check_rendered(html))

    # --- helpers ---
    def _prep_draft(self, n=1, write=True):
        self.s.create_campaign("demo", {"audience": {"segment": "x"}, "sendboxes": ["sb-a"]})
        self._sb()
        self.lead = None
        for i in range(n):
            lead, _ = self.s.add_contact({"name": {"full": f"L{i}"}, "identities": {"emails": [{"address": f"l{i}@x.com", "is_primary": True}]}})
            self.lead = self.lead or lead
            self.s.enrich_write(lead, {"identity": {"still_active": "confirmed"},
                                       "hooks": [{"type": "new_listing", "summary": "s", "evidence_url": "https://z/1",
                                                  "observed_date": "2026-07-14", "confidence": 0.9,
                                                  "analysis": {"sensitivity": "public_business"}}],
                                       "writing_brief": {"personalization_confidence": 0.85}}, campaign_slug="demo")
            if write:
                self.s.draft_write(lead, "demo", 1, f"Idea {i}", "Hi",
                                   hooks_used=[{"type": "new_listing", "evidence_url": "https://z/1"}])

    def test_atomic_write_survives_concurrent_writers(self):
        import threading
        from storage.json_adapter import JsonAdapter
        path = os.path.join(self.cdir, "crm", "pipelines.json"); errors = []
        def w(i):
            try:
                for _ in range(30):
                    JsonAdapter._atomic_write(path, '{"i": %d}' % i)
            except Exception as e:  # noqa: BLE001
                errors.append(repr(e))
        ts = [threading.Thread(target=w, args=(i,)) for i in range(8)]
        [t.start() for t in ts]; [t.join() for t in ts]
        self.assertEqual(errors, [])                                    # no shared-temp-file race
        with open(path) as fh:
            json.load(fh)                                              # final file is valid JSON

    def _idx(self):
        return os.path.join(self.cdir, "outputs", now_iso()[:10], "approval_index.json")


class TestOpsLoopFixes(unittest.TestCase):
    """Pre-production ops-loop fixes: hooks merge-not-overwrite, bump dedupe,
    unsubscribe freeze, and the shared draft budget with a new-lead floor."""

    def setUp(self):
        self.cdir = _client()
        self.s = CrmStore(self.cdir)
        import gmail_client as g
        g.save_sendbox(self.cdir, {"slug": "sb-a", "email": "me@gmail.com", "domain": "gmail.com",
                                   "quota_today": 40, "status": "healthy", "imap_uid_cursor": 0})
        self.s.create_campaign("demo", {"audience": {"segment": "x", "personalization": {"no_hook_fallback": "generic_honest_opener"}},
                                        "sendboxes": ["sb-a"]})
        self.lead, _ = self.s.add_contact({"name": {"full": "A"},
                                           "identities": {"emails": [{"address": "a@x.com", "is_primary": True}]}})

    def _hook(self, url, **over):
        h = {"type": "new_listing", "summary": "x", "evidence_url": url, "observed_date": "2026-07-14",
             "confidence": 0.9, "analysis": {"sensitivity": "public_business"}}
        h.update(over)
        return h

    def test_enrich_refresh_merges_hooks_not_overwrites(self):
        self.s.enrich_write(self.lead, {"hooks": [self._hook("https://z/a"), self._hook("https://z/b")],
                                        "writing_brief": {"personalization_confidence": 0.8}})
        # partial refresh submits ONLY the newly-found hook -> prior reserved hooks must survive
        self.s.enrich_write(self.lead, {"hooks": [self._hook("https://z/c")]})
        urls = {h["evidence_url"] for h in self.s.get_contact(self.lead)["enrichment"]["hooks"]}
        self.assertEqual(urls, {"https://z/a", "https://z/b", "https://z/c"})
        # resubmitting a hook unions used_in instead of dropping history
        self.s.draft_write(self.lead, "demo", 1, "Idea", "Hi",
                           hooks_used=[{"type": "new_listing", "evidence_url": "https://z/a"}])
        self.s.enrich_write(self.lead, {"hooks": [self._hook("https://z/a")]})
        ha = next(h for h in self.s.get_contact(self.lead)["enrichment"]["hooks"]
                  if h["evidence_url"] == "https://z/a")
        self.assertIn("demo/step1", ha.get("used_in", []))
        # explicit retirement removes a stale hook (a sold listing)
        self.s.enrich_write(self.lead, {"hooks": [], "retired_hooks": ["https://z/b"]})
        urls = {h["evidence_url"] for h in self.s.get_contact(self.lead)["enrichment"]["hooks"]}
        self.assertEqual(urls, {"https://z/a", "https://z/c"})

    def _sent_step1_days_ago(self, lead, days):
        import datetime as dt
        sa = (dt.datetime.now(dt.timezone.utc).replace(microsecond=0) - dt.timedelta(days=days)).isoformat().replace("+00:00", "Z")
        d = os.path.join(self.cdir, "campaigns", "demo", "sent", sa[:7])
        os.makedirs(d, exist_ok=True)
        with open(os.path.join(d, "sent_log.jsonl"), "a") as fh:
            fh.write(json.dumps({"seq": 1, "ts": sa, "lead_id": lead, "campaign": "demo", "step": 1,
                                 "sendbox": "sb-a", "rfc_message_id": "<x>", "sent_at": sa}) + "\n")

    def test_followups_due_skips_leads_with_pending_bump_draft(self):
        self._sent_step1_days_ago(self.lead, 5)
        self.assertEqual([x["lead_id"] for x in self.s.followups_due("demo")], [self.lead])
        # a bump draft awaiting approval -> the lead is NOT offered again the next day
        self.s.draft_write(self.lead, "demo", 2, "Re: Idea", "new value")
        self.assertEqual(self.s.followups_due("demo"), [])
        # and a second draft for the same (lead, step) is refused outright
        with self.assertRaises(Exception) as ctx:
            self.s.draft_write(self.lead, "demo", 2, "Re: Idea", "another")
        self.assertIn("duplicate_pending_draft", str(ctx.exception))

    def test_unsubscribe_rule_freezes_sequence(self):
        self.s.apply_rules([{"type": "unsubscribe", "contact_id": self.lead, "activity_id": "a-u1"}])
        c = self.s.get_contact(self.lead)
        self.assertEqual(c.get("sequence_state"), "frozen")
        self.assertTrue(self.s.is_suppressed(email="a@x.com"))

    def test_draft_budget_floor_reserves_new_lead_slots(self):
        self.s.create_campaign("tiny", {"audience": {"segment": "x", "personalization": {"no_hook_fallback": "generic_honest_opener"}},
                                        "sendboxes": ["sb-a"], "daily_quota": 3, "new_lead_floor": 2})
        l2, _ = self.s.add_contact({"identities": {"emails": [{"address": "b@x.com", "is_primary": True}]}})
        self.s.draft_write(self.lead, "tiny", 1, "Hi", "one")            # used 1, remaining 2
        with self.assertRaises(Exception) as ctx:                        # bump blocked: remaining <= floor
            self.s.draft_write(l2, "tiny", 2, "Re: Hi", "bump")
        self.assertIn("bump_budget_exhausted", str(ctx.exception))
        self.s.draft_write(l2, "tiny", 1, "Hi2", "two")                  # step-1 still allowed (used 2)
        l3, _ = self.s.add_contact({"identities": {"emails": [{"address": "c@x.com", "is_primary": True}]}})
        self.s.draft_write(l3, "tiny", 1, "Hi3", "three")                # used 3 = quota
        l4, _ = self.s.add_contact({"identities": {"emails": [{"address": "d@x.com", "is_primary": True}]}})
        with self.assertRaises(Exception) as ctx:                        # quota gone for everyone
            self.s.draft_write(l4, "tiny", 1, "Hi4", "four")
        self.assertIn("draft_budget_exhausted", str(ctx.exception))
        # a reply draft is never budget-blocked
        r = self.s.draft_write(l2, "tiny", 2, "Re: Hi", "answering their reply", is_reply=True)
        self.assertTrue(r["draft_id"])


if __name__ == "__main__":
    unittest.main(verbosity=2)
