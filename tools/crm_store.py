#!/usr/bin/env python3
"""crm_store.py — the ONLY sanctioned writer for a client's crm/ collections.

Everything that mutates contacts, accounts, deals, activities, tasks, pipelines,
segments, or suppression goes through here (DESIGN §6/§7; AGENTS.md). It is a small
library (import it) and a CLI (call it). Stdlib-only.

Client addressing: a client's workspace is
  {pipeline_root}/clients/{client_slug}/{business_slug}_{location_slug}/
Pass --client-dir <that path>, or --pipeline <root> --client <slug> to resolve the
single workspace under clients/{slug}/.

CLI (JSON to stdout):
  init-client   --pipeline R --client S [--business B --location L]
  contact add   --json '{...}'         | contact get ID | contact list [--where F,OP,V ...]
  contact merge --loser ID --winner ID
  activity log  --json '{...}'
  pipeline get  | pipeline set --file pipelines.json | pipeline ensure-default
  suppress add  --kind email --value X --reason unsubscribe [--tier client|global] [--tag test_fixture]
  suppress check --email X [--phone Y]
  deal create   --contact ID --stage new_reply [--json '{...}'] | deal move --id ID --stage X --evidence ACT
  task add      --json '{...}' | task done --id ID
  reserve       --sendbox sb-a --day 2026-07-16 --cap 40
  apply-rules   --event TYPE --contact ID [--activity ID] | apply-rules --events file.json
  validate      [--rebuild-index]              (schema check; rebuilds the identity index —
                                                run once when migrating a Phase-0 install, DESIGN §22 R3)
  reset-client  --client-dir DIR --confirm     (test helper; DESIGN §17)
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import glob

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from storage import (  # noqa: E402
    get_adapter, Cond, StorageError, new_ulid, now_iso, today_str,
    normalize_email, normalize_phone, normalize_social,
)
from storage.json_adapter import JsonAdapter  # noqa: E402


DEFAULT_PIPELINES = {
    "pipelines": [{"id": "default_sales", "stages": [
        {"id": "new_reply", "probability": 0.10, "sla_days": 1},
        {"id": "engaged", "probability": 0.25, "sla_days": 7},
        {"id": "meeting_booked", "probability": 0.50, "sla_days": 7},
        {"id": "proposal_sent", "probability": 0.70, "sla_days": 10},
        {"id": "won"}, {"id": "lost"},
    ]}],
    "rules": [
        {"id": "r1", "on": "reply_positive", "do": ["create_deal_if_none(stage=new_reply)", "create_task(title=Reply within 4h,due=+4h)", "freeze_sequence"]},
        {"id": "r2", "on": "reply_question", "do": ["create_deal_if_none(stage=engaged)", "freeze_sequence", "draft_reply_for_approval"]},
        {"id": "r3", "on": "reply_negative|remove_intent", "do": ["suppress(contact)", "freeze_sequence", "close_open_tasks"]},
        {"id": "r4", "on": "stage_age_exceeds_sla", "do": ["create_task(nudge)", "flag_in_report"]},
        {"id": "r5", "on": "deal_won", "do": ["set_lifecycle(customer)", "enroll_segment(customers)", "create_task(onboarding)"]},
        {"id": "r6", "on": "hard_bounce|unsubscribe", "do": ["suppress(contact)", "close_open_tasks"]},
    ],
}


class CrmStore:
    def __init__(self, client_dir: str):
        self.client_dir = os.path.abspath(client_dir)
        self.crm_root = os.path.join(self.client_dir, "crm")
        self.a: JsonAdapter = get_adapter(self.client_dir)  # type: ignore

    # --- scaffold -------------------------------------------------------------

    def init_tree(self) -> None:
        for sub in ("contacts", "accounts", "deals", "activities", "tasks",
                    "segments", "reports"):
            os.makedirs(os.path.join(self.crm_root, sub), exist_ok=True)
        for sub in ("sendboxes", "lists", "campaigns", "approvals", "analytics",
                    "inbox_sync", "integrations/providers", "outputs"):
            os.makedirs(os.path.join(self.client_dir, sub), exist_ok=True)
        self.ensure_default_pipelines()

    # --- pipelines (a config file under crm/, written here for auditability) ---

    def pipelines_path(self) -> str:
        return os.path.join(self.crm_root, "pipelines.json")

    def get_pipelines(self) -> dict:
        p = self.pipelines_path()
        if os.path.isfile(p):
            with open(p, "r", encoding="utf-8") as fh:
                return json.load(fh)
        return {}

    def set_pipelines(self, obj: dict) -> None:
        JsonAdapter._atomic_write(self.pipelines_path(), json.dumps(obj, ensure_ascii=False, indent=2))

    def ensure_default_pipelines(self) -> dict:
        if not os.path.isfile(self.pipelines_path()):
            self.set_pipelines(DEFAULT_PIPELINES)
        return self.get_pipelines()

    # --- segments (saved filters; a config file under crm/) -------------------

    def segments_path(self) -> str:
        return os.path.join(self.crm_root, "segments.json")

    def get_segments(self) -> dict:
        p = self.segments_path()
        if os.path.isfile(p):
            with open(p, "r", encoding="utf-8") as fh:
                return json.load(fh)
        return {"segments": []}

    def set_segment(self, seg: dict) -> dict:
        data = self.get_segments()
        segs = [s for s in data.get("segments", []) if s.get("id") != seg["id"]]
        segs.append(seg)
        data["segments"] = segs
        JsonAdapter._atomic_write(self.segments_path(), json.dumps(data, ensure_ascii=False, indent=2))
        return seg

    def resolve_segment(self, seg_id: str) -> list:
        """Contacts matching a saved segment's `where`, excluding merged tombstones,
        do_not_contact, and any suppressed identity. Segment `where` is a list of
        [field, op, value] using the flat Cond DSL (§6)."""
        seg = next((s for s in self.get_segments().get("segments", []) if s.get("id") == seg_id), None)
        if not seg:
            raise StorageError(f"segment {seg_id!r} not found")
        where = [Cond(*c) for c in seg.get("where", [])]
        out = []
        for c in self.a.query("contacts", where=where):
            if (c.get("merge") or {}).get("status") == "merged":
                continue
            if c.get("lifecycle_stage") == "do_not_contact":
                continue
            if (c.get("merge") or {}).get("status") != "merged" and self._contact_suppressed(c):
                continue
            out.append(c)
        return out

    def _contact_suppressed(self, contact: dict) -> bool:
        for kind, val in self._identity_pairs(contact):
            if kind == "email" and self.is_suppressed(email=val):
                return True
            if kind == "phone" and self.is_suppressed(phone=val):
                return True
            if kind == "social" and self.is_suppressed(socials=[val]):
                return True
        return False

    # --- campaigns (config file per campaign, outside crm/) -------------------

    def campaign_dir(self, slug: str) -> str:
        return os.path.join(self.client_dir, "campaigns", slug)

    def campaign_config_path(self, slug: str) -> str:
        return os.path.join(self.campaign_dir(slug), "campaign_config.json")

    def get_campaign(self, slug: str) -> dict | None:
        p = self.campaign_config_path(slug)
        if os.path.isfile(p):
            with open(p, "r", encoding="utf-8") as fh:
                return json.load(fh)
        return None

    def list_campaigns(self) -> list:
        root = os.path.join(self.client_dir, "campaigns")
        out = []
        if os.path.isdir(root):
            for slug in sorted(os.listdir(root)):
                cfg = self.get_campaign(slug)
                if cfg:
                    out.append(cfg)
        return out

    _GOAL_TYPES = {"book_meeting", "get_reply", "direct_sale", "reactivation", "nurture_upsell", "event_invite"}

    def create_campaign(self, slug: str, config: dict) -> dict:
        goal = config.get("goal", {})
        gt = goal.get("goal_type")
        if gt and gt not in self._GOAL_TYPES:
            raise StorageError(f"goal_type {gt!r} not in {sorted(self._GOAL_TYPES)}")
        cfg = {
            "schema_version": 1, "campaign_slug": slug,
            "goal": {"goal_type": gt or "get_reply", "objective": "", "offer": "",
                     "value_proposition": "", "proof_points": [],
                     "cta": {"type": "reply_yes", "text": ""},
                     "success_event": {"on": "reply_positive", "create_deal_stage": "new_reply"}},
            "audience": {"segment": "", "personalization": {"required_hook_types": [],
                         "min_confidence": 0.7, "no_hook_fallback": "generic_honest_opener"}},
            "sequence": [{"step": 1, "intent": "hook + offer, one CTA", "tracking": "plain_text"},
                         {"step": 2, "gap_days": 4, "intent": "deliver new value"},
                         {"step": 3, "gap_days": 5, "intent": "social proof"},
                         {"step": 4, "gap_days": 7, "intent": "breakup"}],
            "sendboxes": [], "daily_quota": 40, "approval_mode": "manual_all",
            "channel_strategy": "email_first",
            "min_days_between_touches_across_campaigns": 7,
            "guardrails": {"banned_claims": ["guarantees"], "no_fake_re": True},
            "status": "active",
        }
        _deep_update(cfg, config)
        cfg["campaign_slug"] = slug
        os.makedirs(os.path.join(self.campaign_dir(slug), "queue", "enriched"), exist_ok=True)
        os.makedirs(os.path.join(self.campaign_dir(slug), "outbox", "pending_approval"), exist_ok=True)
        os.makedirs(os.path.join(self.campaign_dir(slug), "outbox", "approved"), exist_ok=True)
        os.makedirs(os.path.join(self.campaign_dir(slug), "history"), exist_ok=True)
        JsonAdapter._atomic_write(self.campaign_config_path(slug), json.dumps(cfg, ensure_ascii=False, indent=2))
        return cfg

    # --- enrich queue (JIT buffer) -------------------------------------------

    def _enrich_queue_path(self, slug: str) -> str:
        return os.path.join(self.campaign_dir(slug), "queue", "enrich_queue.jsonl")

    def _queued_or_sent_leads(self, slug: str) -> set:
        """lead_ids already queued or already sent in THIS campaign (any step)."""
        out = set()
        qp = self._enrich_queue_path(slug)
        if os.path.isfile(qp):
            for line in open(qp, "r", encoding="utf-8"):
                line = line.strip()
                if line:
                    try:
                        out.add(json.loads(line)["lead_id"])
                    except (ValueError, KeyError):
                        pass
        for p in self._all_sent_logs(only_campaign=slug):
            for line in open(p, "r", encoding="utf-8"):
                line = line.strip()
                if line:
                    try:
                        out.add(json.loads(line)["lead_id"])
                    except (ValueError, KeyError):
                        pass
        return out

    def _all_sent_logs(self, only_campaign: str | None = None):
        root = os.path.join(self.client_dir, "campaigns")
        files = []
        if os.path.isdir(root):
            for camp in os.listdir(root):
                if only_campaign and camp != only_campaign:
                    continue
                base = os.path.join(root, camp, "sent")
                if os.path.isdir(base):
                    for month in sorted(os.listdir(base)):
                        fp = os.path.join(base, month, "sent_log.jsonl")
                        if os.path.isfile(fp):
                            files.append(fp)
        return files

    def _last_touch_other_campaign(self, lead_id: str, this_campaign: str) -> str:
        """Most recent sent_at for this lead from ANY OTHER campaign, or ''."""
        latest = ""
        for p in self._all_sent_logs():
            for line in open(p, "r", encoding="utf-8"):
                line = line.strip()
                if not line:
                    continue
                try:
                    r = json.loads(line)
                except ValueError:
                    continue
                if r.get("lead_id") == lead_id and r.get("campaign") != this_campaign:
                    sa = r.get("sent_at", "")
                    if sa > latest:
                        latest = sa
        return latest

    def queue_campaign(self, slug: str, limit: int = 100) -> dict:
        """Populate the enrich queue (JIT buffer) for a campaign from its audience segment,
        applying the don't-double-touch guards. Idempotent-ish: never re-queues a lead already
        queued or sent in this campaign."""
        cfg = self.get_campaign(slug)
        if not cfg:
            raise StorageError(f"campaign {slug!r} not found")
        seg_id = cfg.get("audience", {}).get("segment")
        if not seg_id:
            raise StorageError(f"campaign {slug!r} has no audience.segment")
        min_days = int(cfg.get("min_days_between_touches_across_campaigns", 7))
        cutoff = _iso_days_ago(min_days)
        already = self._queued_or_sent_leads(slug)
        candidates = self.resolve_segment(seg_id)
        added, skipped = 0, {"already_in_campaign": 0, "recently_touched_elsewhere": 0,
                             "in_active_sequence": 0, "no_email": 0}
        qp = self._enrich_queue_path(slug)
        os.makedirs(os.path.dirname(qp), exist_ok=True)
        with open(qp, "a", encoding="utf-8") as qf:
            for c in candidates:
                if added >= limit:
                    break
                lead_id = c["id"]
                if lead_id in already:
                    skipped["already_in_campaign"] += 1
                    continue
                # email_first campaigns need a real found email (no guessing in MVP)
                if cfg.get("channel_strategy", "email_first") == "email_first" and \
                        not [e for e in c.get("identities", {}).get("emails", []) if e.get("address")]:
                    skipped["no_email"] += 1
                    continue
                if c.get("sequence_state") == "frozen":
                    skipped["in_active_sequence"] += 1
                    continue
                last = self._last_touch_other_campaign(lead_id, slug)
                if last and last >= cutoff:
                    skipped["recently_touched_elsewhere"] += 1
                    continue
                qf.write(json.dumps({"lead_id": lead_id, "campaign": slug, "queued_at": now_iso(),
                                     "status": "queued", "step": 1}) + "\n")
                added += 1
        return {"campaign": slug, "queued": added, "skipped": skipped, "segment": seg_id}

    # --- validation / migration ----------------------------------------------

    def validate(self, rebuild_index: bool = False) -> dict:
        """Validate every crm/ record against the Stage 7 minimum schema and, when
        rebuild_index is set, rebuild contact_identities.jsonl from the contacts on disk.
        This is the migration for a workspace whose records were created by the Phase-0
        direct-write path (before the identity reverse index existed): without it,
        find_by_identity/dedupe would miss those contacts (DESIGN §22 R3)."""
        report = {"contacts": 0, "problems": [], "index_rebuilt": False, "identities_indexed": 0}
        req = ("id", "schema_version", "created_at", "updated_at")
        contacts_dir = os.path.join(self.crm_root, "contacts")
        contacts = []
        if os.path.isdir(contacts_dir):
            for name in sorted(os.listdir(contacts_dir)):
                if not name.endswith(".json"):
                    continue
                path = os.path.join(contacts_dir, name)
                try:
                    with open(path, "r", encoding="utf-8") as fh:
                        rec = json.load(fh)
                except (OSError, ValueError) as e:
                    report["problems"].append(f"{name}: unreadable ({e.__class__.__name__})")
                    continue
                report["contacts"] += 1
                for k in req:
                    if not rec.get(k):
                        report["problems"].append(f"{name}: missing/empty {k}")
                if rec.get("id") and rec["id"] != name[:-5]:
                    report["problems"].append(f"{name}: id {rec['id']!r} != filename")
                contacts.append(rec)
        for coll in ("accounts", "deals"):
            d = os.path.join(self.crm_root, coll)
            if os.path.isdir(d):
                for name in sorted(os.listdir(d)):
                    if name.endswith(".json"):
                        try:
                            with open(os.path.join(d, name), "r", encoding="utf-8") as fh:
                                json.load(fh)
                        except (OSError, ValueError) as e:
                            report["problems"].append(f"{coll}/{name}: unreadable ({e.__class__.__name__})")
        if rebuild_index:
            idx_path = os.path.join(self.crm_root, "contact_identities.jsonl")
            tmp = idx_path + ".rebuild.tmp"
            n = 0
            with open(tmp, "w", encoding="utf-8") as fh:
                seq = 0
                for rec in contacts:
                    if (rec.get("merge") or {}).get("status") == "merged":
                        continue  # tombstones don't own live identities
                    for kind, val in self._identity_pairs(rec):
                        seq += 1
                        fh.write(json.dumps({"seq": seq, "ts": now_iso(), "kind": kind,
                                             "value": val, "contact_id": rec["id"], "removed": False}) + "\n")
                        n += 1
            os.replace(tmp, idx_path)
            self.a._identity_cache = None  # force a rebuild on next lookup
            self.a._identity_cache_sig = None
            report["index_rebuilt"] = True
            report["identities_indexed"] = n
        return report

    # --- contacts + identity + merge -----------------------------------------

    def resolve(self, lead_id: str) -> str:
        """Follow merge chains to the surviving contact id."""
        seen = set()
        cur = lead_id
        while cur and cur not in seen:
            seen.add(cur)
            rec = self.a.get("contacts", cur)
            if not rec:
                return cur
            merged_into = (rec.get("merge") or {}).get("merged_into")
            if (rec.get("merge") or {}).get("status") == "merged" and merged_into:
                cur = merged_into
            else:
                return cur
        return cur

    def get_contact(self, lead_id: str) -> dict | None:
        rec = self.a.get("contacts", lead_id)
        if rec and (rec.get("merge") or {}).get("status") == "merged":
            return self.a.get("contacts", self.resolve(lead_id))
        return rec

    def _identity_pairs(self, contact: dict):
        ids = contact.get("identities", {})
        for e in ids.get("emails", []) or []:
            v = normalize_email(e.get("address", ""))
            if v:
                yield ("email", v)
        for p in ids.get("phones", []) or []:
            v = normalize_phone(p.get("number", ""))
            if v:
                yield ("phone", v)
        for _, url in (ids.get("socials", {}) or {}).items():
            v = normalize_social(url or "")
            if v:
                yield ("social", v)

    def add_contact(self, fields: dict) -> tuple[str, str]:
        """Create or match a contact. Returns (lead_id, outcome) where outcome is
        'created' or 'matched'. Auto-merges on an exact identity hit."""
        with self.a._lock("contacts_add"):
            # dedupe against existing identities
            for kind, val in self._identity_pairs(fields):
                existing = self.a.find_by_identity(kind, val)
                if existing:
                    return self.resolve(existing), "matched"
            lead_id = fields.get("id") or new_ulid("c_")
            rec = _contact_skeleton(lead_id)
            _merge_into_contact(rec, fields)
            self.a.put("contacts", lead_id, rec)
            for kind, val in self._identity_pairs(rec):
                self.a.register_identity(kind, val, lead_id)
            return lead_id, "created"

    def set_contact(self, lead_id: str, patch: dict) -> dict:
        lead_id = self.resolve(lead_id)
        def mut(rec):
            _merge_into_contact(rec, patch)
            return rec
        rec = self.a.update("contacts", lead_id, mut)
        for kind, val in self._identity_pairs(rec):
            if self.a.find_by_identity(kind, val) != lead_id:
                self.a.register_identity(kind, val, lead_id)
        return rec

    def merge(self, loser_id: str, winner_id: str) -> dict:
        loser_id, winner_id = self.resolve(loser_id), self.resolve(winner_id)
        if loser_id == winner_id:
            return self.a.get("contacts", winner_id)
        loser = self.a.get("contacts", loser_id)
        if not loser:
            raise StorageError(f"loser {loser_id} not found")
        def mut(win):
            li = win.setdefault("identities", {})
            lo = loser.get("identities", {})
            have_e = {normalize_email(x.get("address")) for x in li.setdefault("emails", [])}
            for e in lo.get("emails", []) or []:
                if normalize_email(e.get("address")) not in have_e:
                    li["emails"].append(e); have_e.add(normalize_email(e.get("address")))
            have_p = {normalize_phone(x.get("number")) for x in li.setdefault("phones", [])}
            for pp in lo.get("phones", []) or []:
                if normalize_phone(pp.get("number")) not in have_p:
                    li["phones"].append(pp); have_p.add(normalize_phone(pp.get("number")))
            for k, v in (lo.get("socials", {}) or {}).items():
                if v and not (li.setdefault("socials", {}).get(k)):
                    li["socials"][k] = v
            # union opted_out channel statuses (safety: keep the more-restrictive)
            for ch, cval in (loser.get("channels", {}) or {}).items():
                if cval.get("status") in ("opted_out", "bounced"):
                    win.setdefault("channels", {}).setdefault(ch, {})["status"] = cval["status"]
            return win
        win = self.a.update("contacts", winner_id, mut)
        # tombstone the loser
        self.a.update("contacts", loser_id, lambda r: {**r, "merge": {"status": "merged", "merged_into": winner_id}})
        # re-point loser identities to the winner
        for kind, val in self._identity_pairs(loser):
            self.a.register_identity(kind, val, winner_id)
        self.log_activity("merged", winner_id, summary=f"merged {loser_id} into {winner_id}", by="agent",
                          ref={"path": loser_id})
        return win

    # --- activities / tasks / deals ------------------------------------------

    def log_activity(self, type: str, contact_id: str, summary: str = "", by: str = "agent",
                     deal_id=None, ref=None) -> dict:
        return self.a.append("activities", {
            "id": new_ulid("act_"), "contact_id": self.resolve(contact_id) if contact_id else None,
            "deal_id": deal_id, "type": type, "summary": summary, "ref": ref or {}, "by": by,
        })

    def add_task(self, title: str, contact_id=None, deal_id=None, due_at="", created_by="agent",
                 guard_key="") -> dict:
        rec = {"id": new_ulid(""), "contact_id": self.resolve(contact_id) if contact_id else None,
               "deal_id": deal_id, "title": title, "due_at": due_at, "status": "open",
               "created_by": created_by, "guard_key": guard_key}
        return self.a.append("tasks", rec)

    def open_tasks_for(self, contact_id: str) -> list:
        cid = self.resolve(contact_id)
        latest = {}
        for t in self.a.read_log("tasks"):
            latest[t["id"]] = t  # last write wins per id (status updates appended)
        return [t for t in latest.values() if t.get("contact_id") == cid and t.get("status") == "open"]

    def close_tasks(self, contact_id: str, by: str = "rule") -> int:
        n = 0
        for t in self.open_tasks_for(contact_id):
            self.a.append("tasks", {**t, "status": "cancelled"})
            n += 1
        return n

    def create_deal(self, contact_id: str, stage: str, pipeline="default_sales", by="rule",
                    evidence_activity_id="", extra=None) -> dict:
        cid = self.resolve(contact_id)
        did = new_ulid("d_")
        rec = {"id": did, "schema_version": 1, "name": (extra or {}).get("name", ""),
               "contact_ids": [cid], "account_id": (extra or {}).get("account_id", ""),
               "pipeline": pipeline, "stage": stage, "value": (extra or {}).get("value", 0),
               "currency": (extra or {}).get("currency", "USD"),
               "probability": self._stage_prob(pipeline, stage),
               "expected_close": "", "source_campaign": (extra or {}).get("source_campaign", ""),
               "stage_history": [{"stage": stage, "at": now_iso(), "by": by, "evidence_activity_id": evidence_activity_id}],
               "status": "open", "lost_reason": None, "next_action": {"task_id": None}}
        self.a.put("deals", did, rec)
        self.log_activity("stage_change", cid, summary=f"deal {did} created at {stage}", by=by, deal_id=did,
                          ref={"path": evidence_activity_id})
        return rec

    def open_deal_for(self, contact_id: str) -> dict | None:
        cid = self.resolve(contact_id)
        for d in self.a.query("deals", where=[Cond("status", "=", "open")]):
            if cid in d.get("contact_ids", []):
                return d
        return None

    def move_deal(self, deal_id: str, stage: str, evidence_activity_id: str, by="rule") -> dict:
        def mut(d):
            d["stage"] = stage
            d["probability"] = self._stage_prob(d.get("pipeline", "default_sales"), stage)
            d.setdefault("stage_history", []).append(
                {"stage": stage, "at": now_iso(), "by": by, "evidence_activity_id": evidence_activity_id})
            if stage == "won":
                d["status"] = "won"
            elif stage == "lost":
                d["status"] = "lost"
            return d
        return self.a.update("deals", deal_id, mut)

    def _stage_prob(self, pipeline: str, stage: str) -> float:
        for p in self.get_pipelines().get("pipelines", []):
            if p.get("id") == pipeline:
                for s in p.get("stages", []):
                    if s.get("id") == stage:
                        return float(s.get("probability", 0.0))
        return 0.0

    # --- suppression ----------------------------------------------------------

    def _global_suppression_path(self) -> str:
        # pipeline root = two levels above clients/{slug}/{workspace}
        d = self.client_dir
        for _ in range(6):
            cand = os.path.join(d, "suppression", "global_suppression.jsonl")
            if os.path.isdir(os.path.join(d, "clients")) or os.path.isfile(os.path.join(d, "storage_config.json")):
                return cand
            d = os.path.dirname(d)
        return os.path.join(self.client_dir, "..", "..", "..", "suppression", "global_suppression.jsonl")

    def suppress(self, kind: str, value: str, reason: str, tier="client", scope=None,
                 source_activity_id="", by="rule", tags=None) -> dict:
        norm = _normalize_by_kind(kind, value)
        rec = {"tier": tier, "match": {"kind": kind, "value": norm}, "reason": reason,
               "scope": scope or ("all_clients" if tier == "global" else os.path.basename(os.path.dirname(self.client_dir))),
               "source_activity_id": source_activity_id, "added_by": by, "tags": tags or []}
        if tier == "global":
            gp = self._global_suppression_path()
            os.makedirs(os.path.dirname(gp), exist_ok=True)
            rec = {**rec, "seq": _append_jsonl_seq(gp), "ts": now_iso()}
            with open(gp, "a", encoding="utf-8") as fh:
                fh.write(json.dumps(rec, ensure_ascii=False) + "\n")
            return rec
        return self.a.append("suppression", rec)

    def suppress_contact(self, contact_id: str, reason: str, source_activity_id="", by="rule") -> int:
        c = self.get_contact(contact_id)
        if not c:
            return 0
        n = 0
        for kind, val in self._identity_pairs(c):
            self.suppress(kind, val, reason, tier="client", source_activity_id=source_activity_id, by=by)
            n += 1
        # also flip email channel to opted_out on unsubscribe/negative
        if reason in ("unsubscribe", "reply_negative", "remove_intent"):
            self.set_contact(contact_id, {"channels": {"email": {"status": "opted_out"}}})
        return n

    def is_suppressed(self, email=None, phone=None, socials=None) -> dict | None:
        """Return the first matching suppression record (client or global tier), else None."""
        wanted = []
        if email:
            wanted.append(("email", normalize_email(email)))
            dom = normalize_email(email).split("@")[-1] if "@" in email else ""
            if dom:
                wanted.append(("domain", dom))
        if phone:
            wanted.append(("phone", normalize_phone(phone)))
        for u in (socials or []):
            wanted.append(("social", normalize_social(u)))
        rows = list(self.a.read_log("suppression"))
        gp = self._global_suppression_path()
        if os.path.isfile(gp):
            for line in open(gp, "r", encoding="utf-8"):
                line = line.strip()
                if line:
                    try:
                        rows.append(json.loads(line))
                    except ValueError:
                        pass
        for r in rows:
            m = r.get("match", {})
            if (m.get("kind"), m.get("value")) in wanted:
                return r
        return None

    # --- quota reservation ----------------------------------------------------

    def reserve(self, sendbox: str, day: str, cap: int) -> str | None:
        return self.a.reserve(sendbox, day, cap)

    # --- deterministic rules engine ------------------------------------------

    def _guard_seen(self, rule_id: str, activity_id: str) -> bool:
        key = f"{rule_id}:{activity_id}"
        for g in self.a.read_log("_rule_guards"):
            if g.get("key") == key:
                return True
        return False

    def _guard_mark(self, rule_id: str, activity_id: str) -> None:
        self.a.append("_rule_guards", {"key": f"{rule_id}:{activity_id}"})

    def apply_rules(self, events: list) -> dict:
        """events: [{type, contact_id, activity_id?, deal_id?}]. Deterministic, idempotent
        via guard keys. Returns {applied:[...], pending:[...]} — pending actions
        (draft_reply_for_approval, flag_in_report) are for the agent, not executed here."""
        pipelines = self.get_pipelines()
        rules = pipelines.get("rules", DEFAULT_PIPELINES["rules"])
        applied, pending = [], []
        for ev in events:
            etype = ev.get("type")
            cid = ev.get("contact_id")
            # A real triggering activity_id makes re-runs idempotent. When the caller
            # supplies none, derive a STABLE key from (type, contact) instead of a random
            # one, so re-running the same event does not double-create deals/tasks.
            aid = ev.get("activity_id") or f"noact:{etype}:{cid}"
            for rule in rules:
                triggers = set(rule.get("on", "").split("|"))
                if etype not in triggers:
                    continue
                if aid and self._guard_seen(rule["id"], aid):
                    continue
                for action in rule.get("do", []):
                    res = self._do_action(action, rule["id"], ev, cid, aid, pending)
                    if res:
                        applied.append({"rule": rule["id"], "action": action, "result": res})
                if aid:
                    self._guard_mark(rule["id"], aid)
        return {"applied": applied, "pending": pending}

    def _do_action(self, action: str, rule_id: str, ev: dict, cid: str, aid: str, pending: list):
        name, args = _parse_action(action)
        if name == "create_deal_if_none":
            if not cid:
                return None
            # atomic check-then-create per contact so two concurrent reply events
            # cannot both create an open deal (TOCTOU); create_deal locks col_deals.
            with self.a._lock(f"deal_contact_{self.resolve(cid)}"):
                if not self.open_deal_for(cid):
                    d = self.create_deal(cid, args.get("stage", "new_reply"), by=f"rule:{rule_id}", evidence_activity_id=aid)
                    return {"deal_id": d["id"]}
            return None
        if name == "create_task":
            due = args.get("due", "")
            return {"task_id": self.add_task(args.get("title", "Follow up"), contact_id=cid,
                                             due_at=_due_to_iso(due), created_by=f"rule:{rule_id}",
                                             guard_key=f"{rule_id}:{aid}")["id"]}
        if name == "freeze_sequence":
            if cid:
                self.set_contact(cid, {"sequence_state": "frozen"})
            return {"frozen": True}
        if name in ("suppress", "suppress(contact)"):
            reason = {"reply_negative": "reply_negative", "remove_intent": "remove_intent",
                      "hard_bounce": "hard_bounce", "unsubscribe": "unsubscribe"}.get(ev.get("type"), "manual")
            return {"suppressed_identities": self.suppress_contact(cid, reason, source_activity_id=aid, by=f"rule:{rule_id}")}
        if name == "close_open_tasks":
            return {"closed": self.close_tasks(cid)}
        if name == "set_lifecycle":
            if cid:
                self.set_contact(cid, {"lifecycle_stage": args.get("_pos", "customer")})
            return {"lifecycle": args.get("_pos", "customer")}
        if name == "enroll_segment":
            if cid:
                c = self.get_contact(cid)
                tags = set(c.get("tags", [])); tags.add(f"segment:{args.get('_pos','customers')}")
                self.set_contact(cid, {"tags": sorted(tags)})
            return {"segment": args.get("_pos", "customers")}
        if name in ("draft_reply_for_approval", "flag_in_report"):
            pending.append({"action": name, "contact_id": cid, "activity_id": aid, "rule": rule_id})
            return None
        return None


# --- helpers ------------------------------------------------------------------

def _contact_skeleton(lead_id: str) -> dict:
    return {
        "id": lead_id, "schema_version": 2, "created_at": now_iso(), "updated_at": now_iso(),
        "name": {"full": "", "first": "", "last": ""}, "account_id": "",
        "identities": {"emails": [], "phones": [], "socials": {"facebook": None, "instagram": None, "linkedin": None, "zalo": None, "x": None}, "website": None},
        "channels": {"email": {"status": "needs_data"}, "sms": {"status": "needs_optin", "mode": "assisted"},
                     "messenger": {"status": "needs_data", "mode": "assisted"}, "zalo": {"status": "needs_data", "mode": "assisted"}},
        "lifecycle_stage": "lead", "tz": "", "tags": [], "custom_fields": {}, "owner": "agency",
        "enrichment": {}, "assigned_sendbox": None, "sequence_state": "active",
        "merge": {"status": "active", "merged_into": None}, "next_action": {"task_id": None},
    }


def _merge_into_contact(rec: dict, patch: dict) -> None:
    for k, v in patch.items():
        if k == "id":
            continue
        if k == "identities" and isinstance(v, dict):
            ids = rec.setdefault("identities", {})
            for e in v.get("emails", []) or []:
                norm = normalize_email(e.get("address", ""))
                if norm and not any(normalize_email(x.get("address")) == norm for x in ids.get("emails", [])):
                    ids.setdefault("emails", []).append({**e, "address": norm})  # store canonical
            for p in v.get("phones", []) or []:
                norm = normalize_phone(p.get("number", ""))
                if norm and not any(normalize_phone(x.get("number")) == norm for x in ids.get("phones", [])):
                    ids.setdefault("phones", []).append({**p, "number": norm})  # store canonical E.164
            for sk, sv in (v.get("socials", {}) or {}).items():
                if sv:
                    ids.setdefault("socials", {})[sk] = sv
            if v.get("website"):
                ids["website"] = v["website"]
        elif isinstance(v, dict) and isinstance(rec.get(k), dict):
            _deep_update(rec[k], v)
        else:
            rec[k] = v


def _deep_update(dst: dict, src: dict) -> None:
    for k, v in src.items():
        if isinstance(v, dict) and isinstance(dst.get(k), dict):
            _deep_update(dst[k], v)
        else:
            dst[k] = v


def _normalize_by_kind(kind: str, value: str) -> str:
    if kind == "email":
        return normalize_email(value)
    if kind == "phone":
        return normalize_phone(value)
    if kind == "social":
        return normalize_social(value)
    return (value or "").strip().lower()


def _parse_action(action: str):
    action = action.strip()
    if "(" not in action:
        return action, {}
    name = action[:action.index("(")]
    inner = action[action.index("(") + 1:action.rindex(")")]
    args = {}
    if inner and "=" not in inner:
        args["_pos"] = inner.strip()
    else:
        for part in inner.split(","):
            if "=" in part:
                k, val = part.split("=", 1)
                args[k.strip()] = val.strip()
    return name, args


def _due_to_iso(due: str) -> str:
    """Support '+4h' / '+2d' relative dues (best-effort; second precision)."""
    import datetime as _dt
    if not due:
        return ""
    if due.startswith("+") and due[-1] in ("h", "d"):
        try:
            n = int(due[1:-1])
        except ValueError:
            return ""
        delta = _dt.timedelta(hours=n) if due[-1] == "h" else _dt.timedelta(days=n)
        return (_dt.datetime.now(_dt.timezone.utc).replace(microsecond=0) + delta).isoformat().replace("+00:00", "Z")
    return due


def _iso_days_ago(days: int) -> str:
    import datetime as _dt
    return (_dt.datetime.now(_dt.timezone.utc).replace(microsecond=0) - _dt.timedelta(days=days)).isoformat().replace("+00:00", "Z")


def _append_jsonl_seq(path: str) -> int:
    n = 0
    if os.path.isfile(path):
        for line in open(path, "r", encoding="utf-8"):
            if line.strip():
                n += 1
    return n + 1


def resolve_client_dir(pipeline: str | None, client: str | None, client_dir: str | None,
                       business: str | None = None, location: str | None = None,
                       create: bool = False) -> str:
    if client_dir:
        return os.path.abspath(client_dir)
    if not (pipeline and client):
        raise SystemExit("need --client-dir, or --pipeline and --client")
    base = os.path.join(os.path.abspath(pipeline), "clients", client)
    if create:
        ws = os.path.join(base, f"{business or 'main'}_{location or 'main'}")
        os.makedirs(ws, exist_ok=True)
        # ensure storage_config at pipeline root
        cfg = os.path.join(os.path.abspath(pipeline), "storage_config.json")
        if not os.path.isfile(cfg):
            os.makedirs(os.path.abspath(pipeline), exist_ok=True)
            with open(cfg, "w", encoding="utf-8") as fh:
                json.dump({"backend": "json"}, fh)
        return ws
    matches = [d for d in glob.glob(os.path.join(base, "*")) if os.path.isdir(d)]
    if not matches:
        raise SystemExit(f"no workspace under {base}; run init-client first")
    if len(matches) > 1:
        raise SystemExit(f"multiple workspaces under {base}; pass --client-dir explicitly")
    return matches[0]


# --- CLI ----------------------------------------------------------------------

def main(argv=None) -> int:
    p = argparse.ArgumentParser(description="OutreachCRM mutation gateway (crm_store)")
    p.add_argument("--pipeline"); p.add_argument("--client"); p.add_argument("--client-dir")
    p.add_argument("--business"); p.add_argument("--location")
    sub = p.add_subparsers(dest="cmd", required=True)

    sub.add_parser("init-client")
    c = sub.add_parser("contact"); c.add_argument("op", choices=["add", "get", "list", "merge"])
    c.add_argument("--json"); c.add_argument("--id"); c.add_argument("--loser"); c.add_argument("--winner")
    c.add_argument("--where", action="append", default=[])
    a = sub.add_parser("activity"); a.add_argument("op", choices=["log"]); a.add_argument("--json", required=True)
    pl = sub.add_parser("pipeline"); pl.add_argument("op", choices=["get", "set", "ensure-default"]); pl.add_argument("--file")
    s = sub.add_parser("suppress"); s.add_argument("op", choices=["add", "check"])
    s.add_argument("--kind"); s.add_argument("--value"); s.add_argument("--reason"); s.add_argument("--tier", default="client")
    s.add_argument("--tag", action="append", default=[]); s.add_argument("--email"); s.add_argument("--phone")
    d = sub.add_parser("deal"); d.add_argument("op", choices=["create", "move"])
    d.add_argument("--contact"); d.add_argument("--stage"); d.add_argument("--id"); d.add_argument("--evidence", default=""); d.add_argument("--json")
    t = sub.add_parser("task"); t.add_argument("op", choices=["add", "done"]); t.add_argument("--json"); t.add_argument("--id")
    r = sub.add_parser("reserve"); r.add_argument("--sendbox", required=True); r.add_argument("--day", required=True); r.add_argument("--cap", type=int, required=True)
    ar = sub.add_parser("apply-rules"); ar.add_argument("--event"); ar.add_argument("--contact"); ar.add_argument("--activity", default=""); ar.add_argument("--events")
    rc = sub.add_parser("reset-client"); rc.add_argument("--confirm", action="store_true")
    va = sub.add_parser("validate"); va.add_argument("--rebuild-index", action="store_true")
    cm = sub.add_parser("campaign"); cm.add_argument("op", choices=["create", "get", "list", "queue"])
    cm.add_argument("--slug"); cm.add_argument("--json"); cm.add_argument("--limit", type=int, default=100)
    sg = sub.add_parser("segment"); sg.add_argument("op", choices=["set", "get", "resolve", "list"])
    sg.add_argument("--json"); sg.add_argument("--id")

    args = p.parse_args(argv)

    if args.cmd == "init-client":
        cdir = resolve_client_dir(args.pipeline, args.client, args.client_dir, args.business, args.location, create=True)
        store = CrmStore(cdir); store.init_tree()
        return _out({"ok": True, "client_dir": cdir, "pipelines": "default_sales"})

    if args.cmd == "reset-client":
        cdir = resolve_client_dir(args.pipeline, args.client, args.client_dir)
        if not args.confirm:
            return _out({"error": "reset-client requires --confirm"}, code=2)
        return _out(_reset_client(cdir))

    cdir = resolve_client_dir(args.pipeline, args.client, args.client_dir)
    store = CrmStore(cdir)

    if args.cmd == "validate":
        return _out(store.validate(rebuild_index=args.rebuild_index))
    if args.cmd == "campaign":
        if args.op == "create":
            return _out(store.create_campaign(args.slug, json.loads(args.json) if args.json else {}))
        if args.op == "get":
            return _out(store.get_campaign(args.slug) or {"error": "not found"})
        if args.op == "list":
            return _out(store.list_campaigns())
        if args.op == "queue":
            return _out(store.queue_campaign(args.slug, args.limit))
    if args.cmd == "segment":
        if args.op == "set":
            return _out(store.set_segment(json.loads(args.json)))
        if args.op == "get" or args.op == "list":
            return _out(store.get_segments())
        if args.op == "resolve":
            return _out([{"id": c["id"], "name": c.get("name", {}).get("full", "")} for c in store.resolve_segment(args.id)])
    if args.cmd == "contact":
        if args.op == "add":
            lead_id, outcome = store.add_contact(json.loads(args.json))
            return _out({"lead_id": lead_id, "outcome": outcome})
        if args.op == "get":
            return _out(store.get_contact(args.id) or {"error": "not found"})
        if args.op == "list":
            where = [_parse_where(w) for w in args.where]
            return _out(store.a.query("contacts", where=where))
        if args.op == "merge":
            return _out(store.merge(args.loser, args.winner))
    if args.cmd == "activity":
        ev = json.loads(args.json)
        return _out(store.log_activity(ev["type"], ev.get("contact_id"), ev.get("summary", ""),
                                       ev.get("by", "agent"), ev.get("deal_id"), ev.get("ref")))
    if args.cmd == "pipeline":
        if args.op == "get":
            return _out(store.get_pipelines())
        if args.op == "ensure-default":
            return _out(store.ensure_default_pipelines())
        if args.op == "set":
            store.set_pipelines(json.load(open(args.file)))
            return _out({"ok": True})
    if args.cmd == "suppress":
        if args.op == "add":
            return _out(store.suppress(args.kind, args.value, args.reason, tier=args.tier, tags=args.tag, by="human"))
        if args.op == "check":
            hit = store.is_suppressed(email=args.email, phone=args.phone)
            return _out({"suppressed": bool(hit), "match": hit})
    if args.cmd == "deal":
        if args.op == "create":
            extra = json.loads(args.json) if args.json else {}
            return _out(store.create_deal(args.contact, args.stage or "new_reply", by="human", extra=extra))
        if args.op == "move":
            return _out(store.move_deal(args.id, args.stage, args.evidence, by="human"))
    if args.cmd == "task":
        if args.op == "add":
            t = json.loads(args.json)
            return _out(store.add_task(t["title"], t.get("contact_id"), t.get("deal_id"), t.get("due_at", ""), t.get("created_by", "human")))
        if args.op == "done":
            latest = {x["id"]: x for x in store.a.read_log("tasks")}
            if args.id in latest:
                return _out(store.a.append("tasks", {**latest[args.id], "status": "done"}))
            return _out({"error": "task not found"}, code=2)
    if args.cmd == "reserve":
        tok = store.reserve(args.sendbox, args.day, args.cap)
        return _out({"token": tok, "granted": bool(tok), "count": store.a.reservation_count(args.sendbox, args.day)})
    if args.cmd == "apply-rules":
        if args.events:
            events = json.load(open(args.events))
        elif args.event and args.contact:
            events = [{"type": args.event, "contact_id": args.contact, "activity_id": args.activity}]
        else:
            return _out({"error": "need --events or (--event and --contact)"}, code=2)
        return _out(store.apply_rules(events))
    return _out({"error": "unknown command"}, code=2)


def _reset_client(client_dir: str) -> dict:
    """Wipe test-client CRM data + test_fixture suppression + reservation cursors."""
    import shutil
    removed = []
    for sub in ("crm/contacts", "crm/accounts", "crm/deals", "crm/activities", "crm/tasks",
                "crm/.seq", "crm/.locks", "crm/contact_identities.jsonl", "crm/suppression.jsonl",
                "sendboxes/_reservations", "lists", "campaigns", "inbox_sync", "outputs"):
        p = os.path.join(client_dir, sub)
        if os.path.isdir(p):
            shutil.rmtree(p); removed.append(sub)
        elif os.path.isfile(p):
            os.remove(p); removed.append(sub)
    CrmStore(client_dir).init_tree()
    return {"reset": True, "client_dir": client_dir, "removed": removed}


def _parse_where(spec: str) -> Cond:
    parts = spec.split(",", 2)
    if len(parts) != 3:
        raise SystemExit(f"--where must be field,op,value (got {spec!r})")
    return Cond(parts[0], parts[1], parts[2])


def _out(obj, code: int = 0) -> int:
    print(json.dumps(obj, ensure_ascii=False, indent=2))
    return code


if __name__ == "__main__":
    sys.exit(main())
