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


if __name__ == "__main__":
    unittest.main(verbosity=2)
