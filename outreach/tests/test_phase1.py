#!/usr/bin/env python3
"""Phase 1 regression tests (stdlib unittest — no install needed).

Run:  python3 -m unittest discover -s tests -v
  or: python3 tests/test_phase1.py

Covers the storage adapter contract, crm_store (dedupe/merge/resolve/rules/suppression/
reserve), import_leads (mapping/dedupe/no-email/idempotency/suppression), email_verify
(syntax), and gmail_client (pre-send gate chain + the DSN-before-thread classifier order).
Network is not required; email_verify MX is monkeypatched.
"""

import email
import json
import os
import sys
import tempfile
import unittest

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(ROOT, "tools"))


def now_short():
    import datetime as _dt
    return _dt.datetime.now(_dt.timezone.utc).strftime("%Y-%m-%d")

from storage import get_adapter, Cond, new_ulid, normalize_email, normalize_phone, normalize_social  # noqa: E402
import crm_store  # noqa: E402
from crm_store import CrmStore  # noqa: E402
import import_leads  # noqa: E402
import email_verify  # noqa: E402
import gmail_client  # noqa: E402


def _new_client():
    tmp = tempfile.mkdtemp()
    pipe = os.path.join(tmp, "daily-content-pipeline")
    crm_store.resolve_client_dir(pipe, "max-output", None, "ai", "hcmc", create=True)
    cdir = crm_store.resolve_client_dir(pipe, "max-output", None)
    CrmStore(cdir).init_tree()
    return cdir


class TestStorageAdapter(unittest.TestCase):
    def setUp(self):
        self.cdir = _new_client()
        self.a = get_adapter(self.cdir)

    def test_put_get_stamps_and_guard(self):
        cid = new_ulid("c_")
        self.a.put("contacts", cid, {"id": cid, "lifecycle_stage": "lead"})
        r = self.a.get("contacts", cid)
        self.assertTrue(r["created_at"] and r["updated_at"] and r["schema_version"])
        with self.assertRaises(Exception):
            self.a.put("contacts", cid, {"id": "different"})

    def test_query_ops(self):
        for st in ("lead", "lead", "customer"):
            i = new_ulid("c_"); self.a.put("contacts", i, {"id": i, "lifecycle_stage": st})
        self.assertEqual(len(self.a.query("contacts", where=[Cond("lifecycle_stage", "=", "lead")])), 2)
        self.assertEqual(len(self.a.query("contacts", where=[Cond("lifecycle_stage", "!=", "lead")])), 1)

    def test_append_monotonic_seq_and_ts(self):
        self.a.append("activities", {"type": "imported", "summary": "a"})
        self.a.append("activities", {"type": "email_sent", "summary": "b"})
        rows = self.a.read_log("activities")
        self.assertEqual([r["seq"] for r in rows], [1, 2])
        self.assertTrue(all(r["ts"] for r in rows))

    def test_identity_index(self):
        cid = new_ulid("c_")
        self.a.register_identity("email", normalize_email("A@B.com"), cid)
        self.assertEqual(self.a.find_by_identity("email", "a@b.com"), cid)
        self.assertIsNone(self.a.find_by_identity("email", "z@z.com"))

    def test_reserve_cap(self):
        self.assertTrue(self.a.reserve("sb-a", "2026-07-16", 2))
        self.assertTrue(self.a.reserve("sb-a", "2026-07-16", 2))
        self.assertIsNone(self.a.reserve("sb-a", "2026-07-16", 2))
        self.assertEqual(self.a.reservation_count("sb-a", "2026-07-16"), 2)


class TestNormalization(unittest.TestCase):
    def test_email(self):
        self.assertEqual(normalize_email("  Foo@Bar.COM "), "foo@bar.com")

    def test_phone_e164(self):
        self.assertEqual(normalize_phone("205-369-0520"), "+12053690520")
        self.assertEqual(normalize_phone("+1 (205) 369 0520"), "+12053690520")

    def test_social(self):
        self.assertEqual(normalize_social("https://www.Facebook.com/Foo/?ref=1"), "facebook.com/foo")


class TestCrmStore(unittest.TestCase):
    def setUp(self):
        self.cdir = _new_client()
        self.s = CrmStore(self.cdir)

    def _add(self, **ids):
        return self.s.add_contact({"name": {"full": "X"}, "identities": ids})

    def test_dedupe_email_case_insensitive(self):
        a, o1 = self.s.add_contact({"identities": {"emails": [{"address": "a@b.com"}]}})
        b, o2 = self.s.add_contact({"identities": {"emails": [{"address": "A@B.com"}]}})
        self.assertEqual(o1, "created"); self.assertEqual(o2, "matched"); self.assertEqual(a, b)

    def test_email_optional(self):
        lid, o = self.s.add_contact({"name": {"full": "No Email"}, "identities": {"phones": [{"number": "310-555-0100"}]}})
        self.assertEqual(o, "created")
        c = self.s.get_contact(lid)
        self.assertEqual(c["identities"]["emails"], [])
        self.assertEqual(c["identities"]["phones"][0]["number"], "+13105550100")

    def test_merge_and_resolve(self):
        a, _ = self.s.add_contact({"identities": {"phones": [{"number": "205-369-0520"}]}})
        b, _ = self.s.add_contact({"identities": {"emails": [{"address": "b@x.com"}]}})
        self.s.merge(b, a)
        self.assertEqual(self.s.resolve(b), a)
        self.assertEqual(self.s.get_contact(b)["id"], a)  # resolves through the tombstone

    def test_rules_idempotent(self):
        lid, _ = self.s.add_contact({"identities": {"emails": [{"address": "r@x.com"}]}})
        r1 = self.s.apply_rules([{"type": "reply_positive", "contact_id": lid, "activity_id": "act1"}])
        r2 = self.s.apply_rules([{"type": "reply_positive", "contact_id": lid, "activity_id": "act1"}])
        self.assertTrue(r1["applied"]); self.assertEqual(r2["applied"], [])
        self.assertEqual(len(self.s.a.query("deals")), 1)
        self.assertEqual(self.s.get_contact(lid)["sequence_state"], "frozen")
        self.assertEqual(len(self.s.open_tasks_for(lid)), 1)

    def test_negative_reply_suppresses(self):
        lid, _ = self.s.add_contact({"identities": {"emails": [{"address": "neg@x.com"}]}})
        self.s.apply_rules([{"type": "reply_negative", "contact_id": lid, "activity_id": "a2"}])
        self.assertTrue(self.s.is_suppressed(email="neg@x.com"))
        self.assertEqual(self.s.get_contact(lid)["channels"]["email"]["status"], "opted_out")

    def test_suppression_blocks_domain(self):
        self.s.suppress("domain", "spammy.com", "manual", tier="client")
        self.assertTrue(self.s.is_suppressed(email="anyone@spammy.com"))


class TestImportLeads(unittest.TestCase):
    def setUp(self):
        self.cdir = _new_client()
        self.tmp = tempfile.mkdtemp()
        self.csv = os.path.join(self.tmp, "r.csv")
        with open(self.csv, "w") as fh:
            fh.write('"Email","Full Name","Office Name","Cell Phone"\n')
            fh.write('"a@b.com","Alice","KW","205-369-0520"\n')
            fh.write('"","No Email","Indie","310-555-0100"\n')
            fh.write('"A@B.com","Alice Dup","dup",""\n')

    def test_mapping_inference(self):
        m = import_leads.propose_mapping(["Email", "Full Name", "Office Name", "Cell Phone"])
        self.assertEqual(m["email"], "Email"); self.assertEqual(m["company"], "Office Name")
        self.assertEqual(m["phone"], "Cell Phone")

    def test_import_dedupe_noemail_idempotent(self):
        r = import_leads.do_import(self.cdir, self.csv, "r", None, mx_check=False)["manifest"]
        self.assertEqual(r["contacts_created"], 2)   # alice + no-email
        self.assertEqual(r["contacts_matched_existing"], 1)  # dup alice
        r2 = import_leads.do_import(self.cdir, self.csv, "r", None, mx_check=False)
        self.assertTrue(r2["skipped"])  # idempotency

    def test_import_checks_suppression(self):
        CrmStore(self.cdir).suppress("email", "a@b.com", "unsubscribe", tier="client")
        r = import_leads.do_import(self.cdir, self.csv, "r", None, mx_check=False)["manifest"]
        # both "Alice" rows share the suppressed address -> both suppressed; only no-email is created
        self.assertEqual(r["suppressed_at_import"], 2)
        self.assertEqual(r["contacts_created"], 1)


class TestEmailVerify(unittest.TestCase):
    def test_syntax(self):
        self.assertTrue(email_verify.syntax_ok("a@b.com"))
        self.assertFalse(email_verify.syntax_ok("nope"))
        self.assertFalse(email_verify.syntax_ok("a@@b.com"))
        self.assertFalse(email_verify.syntax_ok("a..b@c.com"))

    def test_check_uses_mx(self):
        orig = email_verify.mx_lookup
        try:
            email_verify.mx_lookup = lambda d, timeout=5.0: ["mx.example.com"] if d == "good.com" else []
            self.assertEqual(email_verify.check("x@good.com")["status"], "mx_ok")
            self.assertEqual(email_verify.check("x@bad.com")["status"], "mx_fail")
        finally:
            email_verify.mx_lookup = orig


class TestGmailPresendAndClassifier(unittest.TestCase):
    def setUp(self):
        self.cdir = _new_client()
        self.s = CrmStore(self.cdir)
        self.lead, _ = self.s.add_contact({"name": {"full": "Binh"},
                                           "identities": {"emails": [{"address": "t@x.com", "is_primary": True, "status": "unverified"}]},
                                           "channels": {"email": {"status": "usable"}}})
        gmail_client.save_sendbox(self.cdir, {"slug": "sb-a", "auth_mode": "app_password", "email": "s@gmail.com",
                                              "domain": "gmail.com", "quota_today": 2, "warmup_stage": "week_1",
                                              "status": "healthy", "imap_uid_cursor": 0, "last_successful_sync_ts": ""})
        os.makedirs(os.path.join(self.cdir, "config"), exist_ok=True)
        with open(os.path.join(self.cdir, "config", "sending_identity.json"), "w") as fh:
            json.dump({"from_name": "Binh at LeadUp",
                       "physical_mailing_address": "123 Main St, Austin, TX 78701"}, fh)

    def _draft(self, **over):
        d = {"id": "draft_1", "lead_id": self.lead, "campaign_slug": "demo", "step": 1, "sendbox": "sb-a",
             "to": "t@x.com", "subject": "Idea", "body_text": "hi", "tracking": "plain_text",
             "status": "approved", "guessed_approved": False}
        d.update(over)
        p = os.path.join(self.cdir, "draft.json")
        with open(p, "w") as fh:
            json.dump(d, fh)
        return p

    def test_happy_dryrun_reserves_nothing(self):
        r = gmail_client.cmd_send(self.cdir, self._draft(), dry_run=True)
        self.assertTrue(r["ok"]); self.assertIn("list_unsubscribe", r)
        # dry-run must NOT consume quota (audit fix)
        self.assertEqual(self.s.a.reservation_count("sb-a", now_short()), 0)

    def test_gate_not_approved(self):
        r = gmail_client.cmd_send(self.cdir, self._draft(status="pending_approval"), dry_run=True)
        self.assertEqual(r["blocker"], "draft_not_approved")

    def test_gate_step1_fake_reply_subject(self):
        r = gmail_client.cmd_send(self.cdir, self._draft(subject="Re: hey"), dry_run=True)
        self.assertEqual(r["blocker"], "step1_subject_looks_like_reply")

    def test_gate_frozen_sequence(self):
        self.s.set_contact(self.lead, {"sequence_state": "frozen"})
        r = gmail_client.cmd_send(self.cdir, self._draft(), dry_run=True)
        self.assertEqual(r["blocker"], "sequence_frozen")

    def test_gate_suppressed(self):
        self.s.suppress("email", "t@x.com", "unsubscribe", tier="client")
        r = gmail_client.cmd_send(self.cdir, self._draft(), dry_run=True)
        self.assertEqual(r["blocker"], "suppressed")

    def test_failed_send_marks_draft_blocked(self):
        """A failed send is never silent: the blocker lands on the draft record (terminal -> blocked)."""
        self.s.suppress("email", "t@x.com", "unsubscribe", tier="client")
        p = self._draft()
        r = gmail_client.cmd_send(self.cdir, p, dry_run=False)
        self.assertEqual(r["blocker"], "suppressed")
        with open(p) as fh:
            d = json.load(fh)
        self.assertEqual(d["status"], "blocked")
        self.assertEqual(d["blocker"], "suppressed")
        self.assertTrue(d["blocked_at"])

    def test_gate_missing_physical_address_blocks_send(self):
        """CAN-SPAM fail-closed: no sending identity / no postal address -> no send."""
        os.remove(os.path.join(self.cdir, "config", "sending_identity.json"))
        r = gmail_client.cmd_send(self.cdir, self._draft(), dry_run=True)
        self.assertFalse(r["ok"]); self.assertEqual(r["blocker"], "missing_physical_address")

    def test_can_spam_footer_appended_to_every_body(self):
        ident = gmail_client.load_sending_identity(self.cdir)
        footer = gmail_client.compliance_footer(ident)
        self.assertIn("123 Main St, Austin, TX 78701", footer)
        sb = gmail_client.get_sendbox(self.cdir, "sb-a")
        draft = {"lead_id": self.lead, "campaign_slug": "demo", "step": 1, "sendbox": "sb-a",
                 "to": "t@x.com", "subject": "Idea", "body_text": "hi", "tracking": "plain_text",
                 "status": "approved"}
        msg = gmail_client.build_mime(sb, draft, "<m@gmail.com>", None, footer=footer)
        body = msg.get_content()
        self.assertIn("123 Main St, Austin, TX 78701", body)
        self.assertIn("unsubscribe", body.lower())
        # footer must come AFTER the draft body, separated by the signature marker
        self.assertLess(body.find("hi"), body.find("-- "))

    def test_classifier_dsn_before_thread(self):
        """The audit blocker: a DSN threaded into our sent message must be a bounce, not a reply."""
        known = {"<orig@gmail.com>": {"lead_id": self.lead, "campaign": "demo"}}
        dsn = email.message_from_string(
            "From: mailer-daemon@googlemail.com\nTo: s@gmail.com\n"
            "In-Reply-To: <orig@gmail.com>\n"
            "Content-Type: multipart/report; report-type=delivery-status; boundary=x\n\n"
            "--x\nContent-Type: text/plain\n\n550 5.1.1 no such user. <orig@gmail.com>\n--x--\n")
        c = gmail_client.classify_message(self.cdir, dsn, "s@gmail.com", known)
        self.assertEqual(c["kind"], "bounce"); self.assertTrue(c["hard"])

    def test_classifier_unsub_alias_empty_body(self):
        c = gmail_client.classify_message(
            self.cdir, email.message_from_string("From: x@y.com\nTo: s+unsub-abc123@gmail.com\nSubject: unsubscribe\n\n"),
            "s@gmail.com", {})
        self.assertEqual(c["kind"], "unsubscribe"); self.assertEqual(c["token"], "abc123")

    def test_classifier_real_reply(self):
        known = {"<orig@gmail.com>": {"lead_id": self.lead, "campaign": "demo"}}
        c = gmail_client.classify_message(
            self.cdir, email.message_from_string("From: t@x.com\nTo: s@gmail.com\nIn-Reply-To: <orig@gmail.com>\n\nyes!"),
            "s@gmail.com", known)
        self.assertEqual(c["kind"], "campaign_reply"); self.assertEqual(c["lead_id"], self.lead)


class TestAuditRegressions(unittest.TestCase):
    """Locks in the fixes for the Phase-1 adversarial code audit (each was an executed bug)."""

    def setUp(self):
        self.cdir = _new_client()
        self.s = CrmStore(self.cdir)

    # --- suppression completeness -------------------------------------------
    def test_suppression_checks_all_identities_and_recipient(self):
        lead, _ = self.s.add_contact({"identities": {"emails": [
            {"address": "primary@x.com", "is_primary": True}, {"address": "secondary@x.com"}]},
            "channels": {"email": {"status": "usable"}}})
        self.s.suppress("email", "secondary@x.com", "unsubscribe", tier="client")
        gmail_client.save_sendbox(self.cdir, {"slug": "sb-a", "email": "s@gmail.com", "domain": "gmail.com",
                                              "quota_today": 5, "status": "healthy", "imap_uid_cursor": 0})
        d = os.path.join(self.cdir, "draft.json")
        with open(d, "w") as fh:
            json.dump({"lead_id": lead, "campaign_slug": "c", "step": 1, "sendbox": "sb-a",
                       "to": "secondary@x.com", "subject": "hi", "body_text": "b", "status": "approved"}, fh)
        r = gmail_client.cmd_send(self.cdir, d, dry_run=True)
        self.assertEqual(r["blocker"], "suppressed")  # a suppressed SECONDARY identity blocks

    def test_recipient_must_be_a_contact_identity(self):
        lead, _ = self.s.add_contact({"identities": {"emails": [{"address": "real@x.com", "is_primary": True}]},
                                      "channels": {"email": {"status": "usable"}}})
        gmail_client.save_sendbox(self.cdir, {"slug": "sb-a", "email": "s@gmail.com", "domain": "gmail.com",
                                              "quota_today": 5, "status": "healthy", "imap_uid_cursor": 0})
        d = os.path.join(self.cdir, "draft.json")
        with open(d, "w") as fh:
            json.dump({"lead_id": lead, "campaign_slug": "c", "step": 1, "sendbox": "sb-a",
                       "to": "attacker@evil.com", "subject": "hi", "body_text": "b", "status": "approved"}, fh)
        r = gmail_client.cmd_send(self.cdir, d, dry_run=True)
        self.assertEqual(r["blocker"], "recipient_not_a_contact_identity")

    # --- rules idempotency without an activity_id ---------------------------
    def test_rules_idempotent_without_activity_id(self):
        lead, _ = self.s.add_contact({"identities": {"emails": [{"address": "r@x.com"}]}})
        self.s.apply_rules([{"type": "reply_positive", "contact_id": lead}])  # no activity_id
        self.s.apply_rules([{"type": "reply_positive", "contact_id": lead}])  # re-run
        self.assertEqual(len(self.s.a.query("deals")), 1)  # not doubled

    # --- phone normalization no longer collapses placeholders ---------------
    def test_placeholder_phones_are_not_a_dedupe_key(self):
        a, o1 = self.s.add_contact({"name": {"full": "A"}, "identities": {"phones": [{"number": "000-000-0000"}]}})
        b, o2 = self.s.add_contact({"name": {"full": "B"}, "identities": {"phones": [{"number": "000-000-0000"}]}})
        self.assertEqual(o1, "created"); self.assertEqual(o2, "created"); self.assertNotEqual(a, b)
        self.assertEqual(self.s.get_contact(a)["identities"]["phones"], [])  # junk phone dropped

    # --- import robustness --------------------------------------------------
    def test_import_ragged_row_does_not_crash(self):
        tmp = tempfile.mkdtemp(); csvp = os.path.join(tmp, "r.csv")
        with open(csvp, "w") as fh:
            fh.write("Email,Full Name,Website\n")
            fh.write("a@b.com,Alice,https://a.com\n")
            fh.write("b@b.com,Bob\n")  # ragged: missing trailing Website -> None
        r = import_leads.do_import(self.cdir, csvp, "r", None, mx_check=False)["manifest"]
        self.assertEqual(r["contacts_created"], 2)  # both imported, no crash

    def test_import_double_header_letters_row(self):
        tmp = tempfile.mkdtemp(); csvp = os.path.join(tmp, "r.csv")
        with open(csvp, "w") as fh:
            fh.write('"A","B","C"\n')                    # the owner's real column-letters row
            fh.write('"Email","Full Name","Cell Phone"\n')
            fh.write('"real@x.com","Real Agent","205-369-0520"\n')
        headers, rows = import_leads._rows_from_file(csvp)
        self.assertIn("Email", headers)                 # real header row detected
        r = import_leads.do_import(self.cdir, csvp, "r", None, mx_check=False)["manifest"]
        self.assertEqual(r["contacts_created"], 1)      # 1 real agent, not a junk 'Email' contact

    # --- DSN bounce with Message-ID in the structured part ------------------
    def test_dsn_message_id_from_rfc822_part(self):
        lead, _ = self.s.add_contact({"identities": {"emails": [{"address": "gone@x.com", "is_primary": True}]}})
        known = {"<orig@gmail.com>": {"lead_id": lead, "campaign": "c"}}
        dsn = email.message_from_string(
            "From: mailer-daemon@googlemail.com\nContent-Type: multipart/report; "
            "report-type=delivery-status; boundary=b\n\n"
            "--b\nContent-Type: text/plain\n\nDelivery to gone@x.com failed permanently.\n"
            "--b\nContent-Type: message/delivery-status\n\n"
            "Final-Recipient: rfc822; gone@x.com\nStatus: 5.1.1\n"
            "--b\nContent-Type: message/rfc822\n\nMessage-ID: <orig@gmail.com>\nSubject: hi\n\nbody\n--b--\n")
        c = gmail_client.classify_message(self.cdir, dsn, "s@gmail.com", known)
        self.assertEqual(c["kind"], "bounce"); self.assertTrue(c["hard"])
        self.assertEqual(c["bounced_message_id"], "<orig@gmail.com>")
        self.assertEqual(c["final_recipient"], "gone@x.com")

    # --- reply with no In-Reply-To still detected via From address ----------
    def test_reply_without_in_reply_to_via_from_fallback(self):
        lead, _ = self.s.add_contact({"identities": {"emails": [{"address": "who@x.com", "is_primary": True}]}})
        resolver = lambda addr: ({"lead_id": self.s.a.find_by_identity("email", normalize_email(addr))}
                                 if self.s.a.find_by_identity("email", normalize_email(addr)) else None)
        msg = email.message_from_string("From: Who <who@x.com>\nTo: s@gmail.com\nSubject: got it\n\nyes")
        c = gmail_client.classify_message(self.cdir, msg, "s@gmail.com", {}, from_resolver=resolver)
        self.assertEqual(c["kind"], "campaign_reply"); self.assertEqual(c["lead_id"], lead)
        self.assertEqual(c["matched_by"], "from_address")

    # --- Cond type-mismatch never crashes -----------------------------------
    def test_query_type_mismatch_no_crash(self):
        i = new_ulid("c_"); self.s.a.put("contacts", i, {"id": i, "schema_version": 2})
        # int field vs str value must not raise
        self.assertEqual(self.s.a.query("contacts", where=[Cond("schema_version", "<", "5")]), [])

    # --- quota release on failure / dry-run ---------------------------------
    def test_reserve_and_release(self):
        t = self.s.a.reserve("sb-x", now_short(), 2)
        self.assertEqual(self.s.a.reservation_count("sb-x", now_short()), 1)
        self.assertTrue(self.s.a.release("sb-x", now_short(), t))
        self.assertEqual(self.s.a.reservation_count("sb-x", now_short()), 0)

    # --- Phase-0 install migration (validate --rebuild-index) ---------------
    def test_validate_rebuilds_identity_index(self):
        # a contact written directly (Phase-0 path) is NOT in the identity index
        import storage as _st
        cid = "c_LEGACY0000000000000000000A"
        os.makedirs(os.path.join(self.cdir, "crm", "contacts"), exist_ok=True)
        rec = {"id": cid, "schema_version": 2, "created_at": _st.now_iso(), "updated_at": _st.now_iso(),
               "name": {"full": "Legacy"}, "identities": {"emails": [{"address": "legacy@x.com"}], "phones": [], "socials": {}},
               "merge": {"status": "active", "merged_into": None}}
        with open(os.path.join(self.cdir, "crm", "contacts", cid + ".json"), "w") as fh:
            json.dump(rec, fh)
        self.assertIsNone(self.s.a.find_by_identity("email", "legacy@x.com"))  # index empty -> dedupe broken
        rep = self.s.validate(rebuild_index=True)
        self.assertTrue(rep["index_rebuilt"]); self.assertEqual(rep["problems"], [])
        self.assertEqual(CrmStore(self.cdir).a.find_by_identity("email", "legacy@x.com"), cid)  # fixed + persisted


if __name__ == "__main__":
    unittest.main(verbosity=2)
