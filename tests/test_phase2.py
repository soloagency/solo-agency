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
    pipe = os.path.join(tmp, "outreach-pipeline")
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

    def test_queue_email_first_skips_no_email(self):
        r = self.s.queue_campaign("demo")
        self.assertEqual(r["queued"], 3)            # a, b, c
        self.assertEqual(r["skipped"]["no_email"], 1)

    def test_queue_idempotent(self):
        self.s.queue_campaign("demo")
        r2 = self.s.queue_campaign("demo")
        self.assertEqual(r2["queued"], 0)
        self.assertEqual(r2["skipped"]["already_in_campaign"], 3)

    def test_queue_skips_recently_touched_by_other_campaign(self):
        # A was emailed by campaign 'other' yesterday -> within the 7-day guard -> skip for 'demo'
        _write_sent(self.cdir, "other", self.a, _yesterday())
        r = self.s.queue_campaign("demo")
        self.assertEqual(r["skipped"]["recently_touched_elsewhere"], 1)
        self.assertEqual(r["queued"], 2)            # b, c only

    def test_queue_skips_frozen_sequence(self):
        self.s.set_contact(self.b, {"sequence_state": "frozen"})  # replied, mid-handling
        r = self.s.queue_campaign("demo")
        self.assertEqual(r["skipped"]["in_active_sequence"], 1)
        self.assertEqual(r["queued"], 2)            # a, c


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
        self.assertTrue(any("no evidence_url" in p for p in r["problems"]))

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
        self.s.create_campaign("demo", {"audience": {"segment": "x"}, "sendboxes": ["sb-a", "sb-b"]})
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

    def test_sticky_sender_on_draft(self):
        self.s.set_contact(self.lead, {"assigned_sendbox": "sb-b"})  # already assigned (even if 'unhealthy')
        r = self.s.draft_write(self.lead, "demo", 2, "Re: Idea", "bump", hooks_used=[{"type": "new_listing", "evidence_url": "https://z/1"}])
        self.assertEqual(r["sendbox"], "sb-b")     # sticky sender wins over rotation for a bump
        self.assertIn("bump_step", r["warnings"])


if __name__ == "__main__":
    unittest.main(verbosity=2)
