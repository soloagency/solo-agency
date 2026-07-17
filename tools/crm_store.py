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
import re
import sys
import glob
import unicodedata

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
        return os.path.join(self.client_dir, "campaigns", _safe_slug(slug, "campaign slug"))

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
        """Resolved lead_ids already queued or already sent in THIS campaign (any step).
        Every id is passed through resolve() so a merged-away id collapses onto the live
        contact — otherwise a merge winner looks un-touched and gets re-queued/re-emailed."""
        out = set()
        qp = self._enrich_queue_path(slug)
        if os.path.isfile(qp):
            with open(qp, "r", encoding="utf-8") as fh:
                for line in fh:
                    line = line.strip()
                    if line:
                        try:
                            out.add(self.resolve(json.loads(line)["lead_id"]))
                        except (ValueError, KeyError):
                            pass
        for p in self._all_sent_logs(only_campaign=slug):
            with open(p, "r", encoding="utf-8") as fh:
                for line in fh:
                    line = line.strip()
                    if line:
                        try:
                            out.add(self.resolve(json.loads(line)["lead_id"]))
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
        """Most recent sent_at for this (resolved) lead from ANY OTHER campaign, or ''."""
        target = self.resolve(lead_id)
        latest = ""
        for p in self._all_sent_logs():
            with open(p, "r", encoding="utf-8") as fh:
                for line in fh:
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        r = json.loads(line)
                    except ValueError:
                        continue
                    if self.resolve(r.get("lead_id")) == target and r.get("campaign") != this_campaign:
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
        candidates = self.resolve_segment(seg_id)
        added, skipped = 0, {"already_in_campaign": 0, "recently_touched_elsewhere": 0,
                             "in_active_sequence": 0, "no_email": 0}
        qp = self._enrich_queue_path(slug)
        os.makedirs(os.path.dirname(qp), exist_ok=True)
        email_first = cfg.get("channel_strategy", "email_first") == "email_first"
        # Lock the read-dedupe-then-append as one unit so two overlapping runs can't both
        # append the same candidate (the already-queued snapshot would otherwise be stale).
        with self.a._lock(f"queue_{_safe_slug(slug, 'campaign slug')}"):
            already = self._queued_or_sent_leads(slug)
            with open(qp, "a", encoding="utf-8") as qf:
                for c in candidates:
                    if added >= limit:
                        break
                    lead_id = self.resolve(c["id"])
                    if lead_id in already:
                        skipped["already_in_campaign"] += 1
                        continue
                    # email_first campaigns need a real, well-formed found email (no guessing in MVP)
                    if email_first and not [e for e in c.get("identities", {}).get("emails", [])
                                            if _valid_email(e.get("address"))]:
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
                    already.add(lead_id)
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

    # --- enrichment (dossier storage + TTL + inheritance) --------------------

    IDENTITY_TTL_DAYS = 90
    HOOK_TTL_DAYS = 10
    NEG_RETRY_DAYS = 30

    def enrich_status(self, contact_id: str, now: str | None = None) -> dict:
        """Does this contact have a fresh-enough dossier? Drives whether the daily run
        enriches, cheaply refreshes hooks, or reuses (cross-campaign inheritance, DESIGN §9.1)."""
        now = now or now_iso()
        c = self.get_contact(contact_id)
        if not c:
            return {"needs": "skip", "reason": "contact_not_found"}
        en = c.get("enrichment") or {}
        ident_fresh = bool(en.get("identity", {}).get("enriched_at")) and \
            en["identity"]["enriched_at"] >= _iso_days_ago_from(self.IDENTITY_TTL_DAYS, now)
        hooks_fresh = bool(en.get("hooks_refreshed_at")) and \
            en["hooks_refreshed_at"] >= _iso_days_ago_from(self.HOOK_TTL_DAYS, now)
        # negative caches (inherited): don't re-burn a dead end before its retry window
        nf = en.get("email_not_found_at")
        if en.get("identity", {}).get("still_active") == "inactive" and ident_fresh:
            return {"needs": "skip", "reason": "known_inactive"}
        if not ident_fresh:
            return {"needs": "enrich", "reason": "identity_stale_or_missing"}
        # a recent email-not-found result is a dead end within its retry window — don't re-chase
        if nf and nf >= _iso_days_ago_from(self.NEG_RETRY_DAYS, now):
            return {"needs": "skip", "reason": "email_not_found_recent"}
        if not hooks_fresh:
            return {"needs": "refresh", "reason": "hooks_stale"}
        return {"needs": "skip", "reason": "dossier_fresh", "confidence_band": en.get("confidence_band")}

    def enrich_due(self, campaign_slug: str, limit: int = 100, now: str | None = None) -> list:
        """Queued leads that still need enrich/refresh (skip the already-fresh ones)."""
        qp = self._enrich_queue_path(campaign_slug)
        out = []
        if os.path.isfile(qp):
            for line in open(qp, "r", encoding="utf-8"):
                line = line.strip()
                if not line:
                    continue
                try:
                    row = json.loads(line)
                except ValueError:
                    continue
                st = self.enrich_status(row["lead_id"], now=now)
                if st["needs"] in ("enrich", "refresh"):
                    out.append({"lead_id": row["lead_id"], "needs": st["needs"], "reason": st["reason"]})
                    if len(out) >= limit:
                        break
        return out

    def enrich_write(self, contact_id: str, dossier: dict, campaign_slug: str | None = None) -> dict:
        """Validate a dossier and store it. HARD validation (DESIGN §9): every USABLE hook must
        carry an evidence_url; a `personal` sensitivity hook is never usable (it only feeds
        do_not_mention); found emails are stored as identities (source 'enrich', never 'guess')."""
        lead_id = self.resolve(contact_id)
        now = now_iso()
        problems = []
        usable_hooks, do_not_mention = [], list((dossier.get("writing_brief") or {}).get("do_not_mention", []))
        for h in dossier.get("hooks", []) or []:
            sens = (h.get("analysis") or {}).get("sensitivity", "public_business")
            if sens == "personal":
                if h.get("summary"):
                    do_not_mention.append(h["summary"])
                continue  # personal-life details never become email copy
            if not _valid_evidence_url(h.get("evidence_url")):
                problems.append(f"hook {h.get('type','?')!r} dropped: evidence_url missing or not a valid http(s) URL")
                continue
            hc = _as_float(h.get("confidence", 0.0), None)
            if hc is None:
                problems.append(f"hook {h.get('type','?')!r}: non-numeric confidence, treated as 0.0")
                hc = 0.0
            usable_hooks.append({"type": h.get("type"), "summary": h.get("summary", ""),
                                 "evidence_url": h["evidence_url"].strip(), "observed_date": h.get("observed_date", ""),
                                 "confidence": hc, "sensitivity": sens,
                                 "used_in": h.get("used_in", [])})
        ident = dossier.get("identity", {}) or {}
        conf = _as_float((dossier.get("writing_brief") or {}).get("personalization_confidence", 0.0), None)
        if conf is None:
            problems.append("personalization_confidence non-numeric, treated as 0.0")
            conf = 0.0
        band = "high" if conf >= 0.7 else ("review_carefully" if conf >= 0.4 else "fallback")
        prev_en = (self.get_contact(lead_id) or {}).get("enrichment") or {}

        # full dossier -> campaign enriched/ (audit); distilled -> contact.enrichment (inherited)
        if campaign_slug:
            d = os.path.join(self.campaign_dir(campaign_slug), "queue", "enriched", today_str(now))
            os.makedirs(d, exist_ok=True)
            JsonAdapter._atomic_write(os.path.join(d, f"{lead_id}.json"),
                                      json.dumps({**dossier, "lead_id": lead_id, "enriched_at": now}, ensure_ascii=False, indent=2))
        # Identity is MERGED against the prior dossier, not rebuilt: a cheap hooks-only refresh
        # (no `identity` block) must not null out still_active/current_company/etc. or reset the
        # 90-day identity TTL — that would silently un-mark a known_inactive contact.
        prev_ident = prev_en.get("identity", {}) or {}
        has_identity = bool(ident)
        merged_ident = {}
        for k in ("still_active", "current_company", "role", "profiles", "evidence"):
            merged_ident[k] = ident.get(k) if ident.get(k) is not None else prev_ident.get(k)
        merged_ident["enriched_at"] = now if has_identity else (prev_ident.get("enriched_at") or now)
        enrichment = {
            "identity": merged_ident,
            "context": dossier.get("context", {}),
            "hooks": usable_hooks, "hooks_refreshed_at": now,
            "writing_brief": {"one_liner": (dossier.get("writing_brief") or {}).get("one_liner", ""),
                              "ranked_angles": (dossier.get("writing_brief") or {}).get("ranked_angles", []),
                              "do_not_mention": do_not_mention, "personalization_confidence": conf},
            "confidence_band": band,
            # negative cache is inherited; set only when this pass explicitly reports a dead end
            "email_not_found_at": now if dossier.get("mark_email_not_found") else prev_en.get("email_not_found_at"),
            "no_verifiable_hook_at": now if (not usable_hooks and dossier.get("mark_no_hook")) else prev_en.get("no_verifiable_hook_at"),
        }
        patch = {"enrichment": enrichment}
        # store any FOUND (never guessed) email/phone the agent discovered
        found = ident.get("channels_found", {}) or {}
        emails = [{"address": e, "source": "enrich", "status": "unverified"} for e in found.get("emails", []) if e]
        phones = [{"number": p, "type": "cell", "source": "enrich"} for p in found.get("phones", []) if p]
        if emails or phones:
            patch["identities"] = {"emails": emails, "phones": phones}
        if ident.get("still_active") == "confirmed":
            patch.setdefault("channels", {}).setdefault("email", {})
        self.set_contact(lead_id, patch)
        return {"lead_id": lead_id, "usable_hooks": len(usable_hooks), "confidence_band": band,
                "do_not_mention": len(do_not_mention), "problems": problems}

    # --- drafts (Stage 6 writes emails; validated + stored here) -------------

    def _sendboxes(self) -> list:
        p = os.path.join(self.client_dir, "sendboxes", "sendboxes.json")
        if os.path.isfile(p):
            with open(p, "r", encoding="utf-8") as fh:
                return json.load(fh).get("sendboxes", [])
        return []

    def _sent_today(self, sendbox_slug: str, day: str) -> int:
        n = 0
        for p in self._all_sent_logs():
            for line in open(p, "r", encoding="utf-8"):
                line = line.strip()
                if not line:
                    continue
                try:
                    r = json.loads(line)
                except ValueError:
                    continue
                if r.get("sendbox") == sendbox_slug and (r.get("sent_at", "") or "")[:10] == day and r.get("rfc_message_id"):
                    n += 1
        return n

    def pick_sendbox(self, campaign_cfg: dict, contact: dict, day: str | None = None) -> str | None:
        """Sticky sender: a contact keeps its assigned box for every bump/reply. Step-1 rotates
        across the campaign's healthy boxes by lowest sent_today/quota_today ratio (DESIGN §8)."""
        if contact.get("assigned_sendbox"):
            return contact["assigned_sendbox"]
        day = day or today_str()
        refs = set(campaign_cfg.get("sendboxes", []))
        # exclude boxes with no capacity today (quota_today<=0) — matching gmail_client's
        # unclamped cap check, so we never pin a draft to a box send will immediately refuse.
        boxes = [b for b in self._sendboxes()
                 if b.get("status") == "healthy" and int(b.get("quota_today", 0)) > 0
                 and (not refs or b["slug"] in refs)]
        if not boxes:
            return None
        def load(b):
            q = int(b.get("quota_today", 1))
            return self._sent_today(b["slug"], day) / q
        boxes.sort(key=lambda b: (load(b), b["slug"]))
        return boxes[0]["slug"]

    def draft_write(self, contact_id: str, campaign_slug: str, step: int, subject: str,
                    body_text: str, hooks_used: list | None = None, body_html: str = "",
                    tracking: str = "plain_text") -> dict:
        """Validate and store a draft in outbox/pending_approval. Rejects a step-1 fake Re:/Fwd:
        subject and any hook_used that isn't in the contact's dossier WITH an evidence_url."""
        lead_id = self.resolve(contact_id)
        contact = self.get_contact(lead_id)
        if not contact:
            raise StorageError("contact_not_found")
        cfg = self.get_campaign(campaign_slug)
        if not cfg:
            raise StorageError(f"campaign {campaign_slug!r} not found")
        step = int(step)
        if step == 1 and re.match(r"^\s*(re|fwd)\s*:", _subject_gate_normalized(subject), re.I):
            raise StorageError("step-1 subject must not begin with Re:/Fwd: (deceptive, CAN-SPAM)")
        # recipient must be a real, well-formed found email (no guessing)
        emails = [e for e in contact.get("identities", {}).get("emails", []) if _valid_email(e.get("address"))]
        primary = next((e for e in emails if e.get("is_primary")), emails[0] if emails else None)
        if not primary:
            raise StorageError("contact has no usable email to draft to")
        # every referenced hook must trace to a dossier hook that carries a VALID evidence_url
        dossier_hooks = (contact.get("enrichment") or {}).get("hooks", [])
        evidence_by_url = {h.get("evidence_url"): h for h in dossier_hooks if _valid_evidence_url(h.get("evidence_url"))}
        clean_hooks = []
        for h in hooks_used or []:
            url = h.get("evidence_url")
            if not url or url not in evidence_by_url:
                raise StorageError(f"hook {h.get('type','?')!r} has no matching evidenced dossier hook — "
                                   "every personalized detail must trace to a dossier hook with an evidence_url")
            # the hook TYPE always comes from the dossier, never the caller — so a real evidence
            # URL can't be relabeled with a fabricated claim (e.g. new_listing -> award_won)
            clean_hooks.append({"type": evidence_by_url[url].get("type"), "evidence_url": url})
        sendbox = self.pick_sendbox(cfg, contact)
        if not sendbox:
            raise StorageError("no healthy sendbox available for this campaign")
        band = (contact.get("enrichment") or {}).get("confidence_band", "review_carefully")
        warnings = []
        if not clean_hooks:
            warnings.append("generic_opener")
        if step > 1:
            warnings.append("bump_step")
        did = new_ulid("draft_")
        now = now_iso()
        draft = {"id": did, "schema_version": 1, "created_at": now, "updated_at": now,
                 "lead_id": lead_id, "campaign_slug": campaign_slug, "step": step,
                 "sendbox": sendbox, "to": primary["address"], "subject": subject,
                 "body_text": body_text, "body_html": body_html,
                 "confidence_band": band, "hooks_used": clean_hooks, "tracking": tracking,
                 "warnings": warnings, "guessed_approved": False,
                 "status": "pending_approval", "decided_at": "", "decided_by": "", "reject_reason": "", "blocker": ""}
        d = os.path.join(self.campaign_dir(campaign_slug), "outbox", "pending_approval", today_str(now))
        os.makedirs(d, exist_ok=True)
        JsonAdapter._atomic_write(os.path.join(d, f"{did}.json"), json.dumps(draft, ensure_ascii=False, indent=2))
        # mark the used hooks so another campaign won't open with the same one
        if clean_hooks:
            used_urls = {h["evidence_url"] for h in clean_hooks}
            def mut(rec):
                for hk in (rec.get("enrichment") or {}).get("hooks", []):
                    if hk.get("evidence_url") in used_urls:
                        tag = f"{campaign_slug}/step{step}"
                        if tag not in hk.setdefault("used_in", []):
                            hk["used_in"].append(tag)
                return rec
            self.a.update("contacts", lead_id, mut)
        return {"draft_id": did, "sendbox": sendbox, "to": primary["address"], "confidence_band": band,
                "warnings": warnings, "path": os.path.join(d, f"{did}.json")}

    def list_pending_drafts(self, campaign_slug: str | None = None) -> list:
        """Every draft still `pending_approval`, deterministically ordered (campaign, created_at,
        id) with a `_path`, so the Approval Report and the approve handler agree on numbering."""
        out = []
        camp_root = os.path.join(self.client_dir, "campaigns")
        camps = [campaign_slug] if campaign_slug else (sorted(os.listdir(camp_root)) if os.path.isdir(camp_root) else [])
        for camp in camps:
            base = os.path.join(self.client_dir, "campaigns", camp, "outbox", "pending_approval")
            if not os.path.isdir(base):
                continue
            for day in sorted(os.listdir(base)):
                dd = os.path.join(base, day)
                if os.path.isdir(dd):
                    for name in sorted(os.listdir(dd)):
                        if not name.endswith(".json"):
                            continue
                        p = os.path.join(dd, name)
                        try:
                            with open(p, "r", encoding="utf-8") as fh:
                                rec = json.load(fh)
                        except (OSError, ValueError):
                            continue
                        if rec.get("status") == "pending_approval":
                            rec["_path"] = p
                            out.append(rec)
        out.sort(key=lambda r: (r.get("campaign_slug", ""), r.get("created_at", ""), r.get("id", "")))
        return out

    # --- Approval Report + chat approval -------------------------------------

    def build_approval(self, campaign_slug: str | None = None, now: str | None = None,
                       number_by_draft: dict | None = None) -> tuple[str, list]:
        """Return (markdown, index). Numbered cards grouped High confidence / Review carefully;
        the index [{n, draft_id, path}] is the number->draft map the approve handler reads.
        `number_by_draft` (draft_id->n from the prior report) keeps a draft's number STABLE
        across re-renders, so approving by a number never silently hits a renumbered draft."""
        now = now or now_iso()
        drafts = self.list_pending_drafts(campaign_slug)
        prior = dict(number_by_draft or {})
        used = set(prior.values())
        nxt = (max(used) + 1) if used else 1
        numbered = []
        for d in drafts:
            n = prior.get(d["id"])
            if n is None:
                n = nxt; nxt += 1
            numbered.append((n, d))
        numbered.sort(key=lambda t: t[0])
        index = [{"n": n, "draft_id": d["id"], "path": d["_path"], "campaign": d["campaign_slug"]}
                 for n, d in numbered]
        high = [(n, d) for n, d in numbered if d.get("confidence_band") == "high"]
        review = [(n, d) for n, d in numbered if d.get("confidence_band") != "high"]

        def card(n, d):
            c = self.get_contact(d["lead_id"]) or {}
            en = c.get("enrichment") or {}
            name = (c.get("name") or {}).get("full", "") or d["to"]
            lines = [f"## {n}. {name} — {d['to']}",
                     f"- **Campaign/step:** {d['campaign_slug']} / step {d['step']}  ·  **Sendbox:** {d['sendbox']}"]
            if d.get("warnings"):
                lines.append(f"- **Flags:** {', '.join(d['warnings'])}")
            hooks = en.get("hooks", [])
            if hooks:
                lines.append("- **Evidence:** " + "  ·  ".join(f"[{h.get('type','hook')}]({h['evidence_url']})"
                             for h in hooks if h.get("evidence_url")))
            lines.append("")
            lines.append(f"**Subject:** {d.get('subject','')}")
            lines.append("")
            lines.append("> " + (d.get("body_text", "").replace("\n", "\n> ")))
            lines.append("")
            return "\n".join(lines)

        md = [f"# Approval Report — {now[:10]}",
              f"{len(drafts)} draft(s) awaiting your approval. Reply in chat: "
              "`approve all` · `approve 1-20, 35` · `reject 7: reason` · `edit 12: ...` · `hold 5`.",
              "", f"## High confidence ({len(high)})",
              "*(verified email + strong evidenced hook)*", ""]
        md += [card(n, d) for n, d in high] or ["*(none)*", ""]
        md += ["", f"## Review carefully ({len(review)})", "*(weak/no hook or fallback opener — read before approving)*", ""]
        md += [card(n, d) for n, d in review] or ["*(none)*", ""]
        return "\n".join(md), index

    def render_approval_report(self, campaign_slug: str | None = None, now: str | None = None) -> dict:
        """Write the Approval Report markdown + index + (best-effort) HTML under outputs/."""
        now = now or now_iso()
        # keep numbers stable across re-renders within the day: reuse the prior index's map
        prior = {e["draft_id"]: e["n"] for e in self._approval_index()}
        md, index = self.build_approval(campaign_slug, now, number_by_draft=prior)
        out_dir = os.path.join(self.client_dir, "outputs", today_str(now))
        os.makedirs(out_dir, exist_ok=True)
        slug = self._client_slug()
        md_path = os.path.join(out_dir, f"{slug}-approval-report.md")
        idx_path = os.path.join(out_dir, "approval_index.json")
        html_path = os.path.join(out_dir, f"{slug}-approval-report.html")
        JsonAdapter._atomic_write(md_path, md)
        JsonAdapter._atomic_write(idx_path, json.dumps({"generated_at": now, "index": index}, ensure_ascii=False, indent=2))
        # operator-only report -> render WITHOUT --client-facing (not scrubbed)
        rendered = False
        try:
            import subprocess
            r = subprocess.run(["python3", os.path.join(os.path.dirname(os.path.abspath(__file__)), "report_renderer.py"),
                                "render", "--input", md_path, "--output-html", html_path,
                                "--title", "Approval Report", "--report-kind", "Approval Report"],
                               capture_output=True, text=True, timeout=60)
            rendered = r.returncode == 0
        except Exception:  # noqa: BLE001
            rendered = False
        return {"drafts": len(index), "md": md_path, "index": idx_path,
                "html": html_path if rendered else None, "html_rendered": rendered}

    def _approval_index(self) -> list:
        p = os.path.join(self.client_dir, "outputs", today_str(), "approval_index.json")
        if os.path.isfile(p):
            try:
                with open(p, "r", encoding="utf-8") as fh:
                    return json.load(fh).get("index", [])
            except (OSError, ValueError):
                return []
        return []

    def _resolve_numbers(self, spec) -> list:
        """'all' | '1-20, 35' | [1,3,5] -> list of {n, draft_id, path} from the last report."""
        idx = self._approval_index()
        by_n = {e["n"]: e for e in idx}
        if spec == "all":
            return idx
        nums = set()
        if isinstance(spec, str):
            for part in spec.replace(" ", "").split(","):
                if not part:
                    continue
                if "-" in part.lstrip("-"):  # a range (not just a leading minus)
                    a, b = part.split("-", 1)
                    a, b = int(a), int(b)
                    if a > b:  # tolerate a reversed range like '20-1'
                        a, b = b, a
                    if b - a > 100_000:  # guard against a fat-fingered/adversarial huge span
                        raise StorageError(f"range {part!r} too wide (max span 100000)")
                    nums.update(range(a, b + 1))
                else:
                    nums.add(int(part))
        else:
            nums = set(int(x) for x in spec)
        return [by_n[n] for n in sorted(nums) if n in by_n]

    def approve_apply(self, actions: dict, by: str = "human") -> dict:
        """Apply the operator's chat decision. `actions` is the agent's parse of the chat message:
        {"approve": "all"|"1-20,35"|[..], "reject":[{"n":7,"reason":"..."}],
         "hold":[5], "edit":[{"n":12,"subject":?,"body_text":?}]}. Numbers reference the last
         Approval Report. Approved drafts move to outbox/approved/ (the send engine's inbox)."""
        result = {"approved": [], "rejected": [], "held": [], "edited": [], "not_found": [],
                  "already_processed": []}
        now = now_iso()
        decided = set()  # numbers with a TERMINAL decision (reject/hold/approve are mutually exclusive)

        def load(path):
            """Return the draft dict, or None if the pending file is already gone (moved/decided) —
            so re-applying a number is a safe no-op, never an uncaught FileNotFoundError."""
            try:
                with open(path, "r", encoding="utf-8") as fh:
                    return json.load(fh)
            except FileNotFoundError:
                return None

        # edits first (so a subsequent approve picks up the edited body); edit is not terminal
        for e in actions.get("edit", []) or []:
            hits = self._resolve_numbers([e["n"]])
            if not hits or (d := load(hits[0]["path"])) is None:
                result["not_found"].append(e["n"]); continue
            entry = hits[0]
            if "subject" in e:
                d["subject"] = e["subject"]
            if "body_text" in e:
                d["body_text"] = e["body_text"]
            d["updated_at"] = now
            JsonAdapter._atomic_write(entry["path"], json.dumps(d, ensure_ascii=False, indent=2))
            result["edited"].append(entry["draft_id"])
        for r in actions.get("reject", []) or []:
            hits = self._resolve_numbers([r["n"]])
            if not hits:
                result["not_found"].append(r["n"]); continue
            if r["n"] in decided:
                continue  # already given a terminal decision earlier in this batch
            entry = hits[0]; d = load(entry["path"])
            if d is None or d.get("status") != "pending_approval":
                result["already_processed"].append(entry["draft_id"]); decided.add(r["n"]); continue
            d.update({"status": "rejected", "decided_at": now, "decided_by": by, "reject_reason": r.get("reason", "")})
            JsonAdapter._atomic_write(entry["path"], json.dumps(d, ensure_ascii=False, indent=2))
            self._approval_log(d, "reject", by, r.get("reason", ""))
            self._learning_log(d, r.get("reason", ""))  # reject reasons feed the writing learning loop
            result["rejected"].append(entry["draft_id"]); decided.add(r["n"])
        for h in self._resolve_numbers(actions.get("hold", []) or []):
            if h["n"] in decided:
                continue
            d = load(h["path"])
            if d is None or d.get("status") != "pending_approval":
                result["already_processed"].append(h["draft_id"]); decided.add(h["n"]); continue
            d.update({"status": "hold", "decided_at": now, "decided_by": by})
            JsonAdapter._atomic_write(h["path"], json.dumps(d, ensure_ascii=False, indent=2))
            result["held"].append(h["draft_id"]); decided.add(h["n"])
        approve_spec = actions.get("approve")
        if approve_spec:
            hits = self._resolve_numbers(approve_spec)
            if approve_spec != "all":  # report explicitly-named numbers that didn't resolve
                by_n = {e["n"]: e for e in self._approval_index()}
                for n in self._requested_numbers(approve_spec):
                    if n not in by_n:
                        result["not_found"].append(n)
            for a in hits:
                if a["n"] in decided:
                    continue
                d = load(a["path"])
                if d is None or d.get("status") != "pending_approval":
                    result["already_processed"].append(a["draft_id"]); decided.add(a["n"]); continue
                d.update({"status": "approved", "decided_at": now, "decided_by": by})
                approved_dir = os.path.join(self.campaign_dir(a["campaign"]), "outbox", "approved")
                os.makedirs(approved_dir, exist_ok=True)
                dest = os.path.join(approved_dir, f"{a['draft_id']}.json")
                # write the approved content back into the pending file, then ATOMICALLY move it to
                # outbox/approved — so a draft can never exist as both 'pending' and 'approved'.
                JsonAdapter._atomic_write(a["path"], json.dumps(d, ensure_ascii=False, indent=2))
                os.replace(a["path"], dest)
                self._approval_log(d, "approve", by, "")
                decided.add(a["n"])
                result["approved"].append({"draft_id": a["draft_id"], "path": dest})
        return result

    def _requested_numbers(self, spec) -> set:
        """The set of individual numbers an approve/reject spec names (for not_found reporting)."""
        nums = set()
        if isinstance(spec, str):
            for part in spec.replace(" ", "").split(","):
                if not part:
                    continue
                if "-" in part.lstrip("-"):
                    a, b = part.split("-", 1); a, b = int(a), int(b)
                    if a > b:
                        a, b = b, a
                    if b - a <= 100_000:
                        nums.update(range(a, b + 1))
                else:
                    nums.add(int(part))
        else:
            nums = set(int(x) for x in spec)
        return nums

    def _approval_log(self, draft, decision, by, reason):
        p = os.path.join(self.client_dir, "approvals", "approval_log.md")
        os.makedirs(os.path.dirname(p), exist_ok=True)
        new = not os.path.isfile(p)
        with open(p, "a", encoding="utf-8") as fh:
            if new:
                fh.write("# Approval Log\n\n| Date | Draft | Campaign/Step | Decision | By | Reason |\n|---|---|---|---|---|---|\n")
            fh.write(f"| {now_iso()} | {draft['id']} | {draft['campaign_slug']}/{draft['step']} | "
                     f"{decision} | {by} | {reason or '—'} |\n")

    def _learning_log(self, draft, reason):
        if not reason:
            return
        p = os.path.join(self.client_dir, "analytics", "learning_log.md")
        os.makedirs(os.path.dirname(p), exist_ok=True)
        new = not os.path.isfile(p)
        with open(p, "a", encoding="utf-8") as fh:
            if new:
                fh.write("# Learning Log\n\n| Date | Source | Signal | Note |\n|---|---|---|---|\n")
            fh.write(f"| {now_iso()} | draft_rejected | {draft['campaign_slug']}/step{draft['step']} | {reason} |\n")

    def _client_slug(self) -> str:
        # {client_slug}/{business_slug}_{location_slug} -> a filesystem-safe report prefix
        return os.path.basename(os.path.dirname(self.client_dir)) or "client"

    # --- follow-ups (silent-lead bumps) --------------------------------------

    def followups_due(self, campaign_slug: str, now: str | None = None) -> list:
        """Contacts due for a bump: sent step N in this campaign, gap_days for step N+1 elapsed,
        no reply (not frozen), sequence not exhausted. Stage 10 drafts these."""
        now = now or now_iso()
        cfg = self.get_campaign(campaign_slug)
        if not cfg:
            raise StorageError(f"campaign {campaign_slug!r} not found")
        seq = cfg.get("sequence", [])
        gap_by_step = {int(s["step"]): int(s.get("gap_days", 0)) for s in seq}
        max_step = max(gap_by_step) if gap_by_step else 1
        # group sent_log by lead: max step + last sent_at
        state = {}
        for p in self._all_sent_logs(only_campaign=campaign_slug):
            with open(p, "r", encoding="utf-8") as fh:
                for line in fh:
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        r = json.loads(line)
                    except ValueError:
                        continue
                    if not r.get("rfc_message_id"):
                        continue
                    # resolve merged ids so one real contact isn't tracked as two due rows
                    lid = self.resolve(r["lead_id"]); st = int(r.get("step", 1)); sa = r.get("sent_at", "")
                    cur = state.get(lid, {"step": 0, "sent_at": ""})
                    if st > cur["step"] or (st == cur["step"] and sa > cur["sent_at"]):
                        state[lid] = {"step": st, "sent_at": sa}
        due = []
        for lid, s in state.items():
            c = self.get_contact(lid)
            if not c or c.get("sequence_state") == "frozen":
                continue  # replied / mid-handling
            next_step = s["step"] + 1
            if next_step > max_step:
                continue  # sequence exhausted (breakup already sent)
            if next_step not in gap_by_step:
                continue  # no defined cadence for this step (non-contiguous config) -> never auto-due
            gap = gap_by_step[next_step]
            if s["sent_at"] and s["sent_at"] <= _iso_days_ago_from(gap, now):
                due.append({"lead_id": lid, "next_step": next_step, "last_step": s["step"],
                            "last_sent_at": s["sent_at"]})
        return due

    # --- Today View + kanban (operator-only) ---------------------------------

    def today_view_data(self, now: str | None = None) -> dict:
        now = now or now_iso()
        tasks = [t for t in self._latest_tasks() if t.get("status") == "open"]
        due_tasks = [t for t in tasks if t.get("due_at") and t["due_at"] <= now]
        deals = self.a.query("deals", where=[Cond("status", "=", "open")])
        sla = []
        stage_sla = {s["id"]: s.get("sla_days") for p in self.get_pipelines().get("pipelines", []) for s in p.get("stages", [])}
        for d in deals:
            hist = d.get("stage_history", [])
            entered = hist[-1]["at"] if hist else d.get("created_at", "")
            sd = stage_sla.get(d.get("stage"))
            if not sd:
                continue
            # a deal with no known entered-time (hand-edited / migrated) is treated as maximally
            # overdue and flagged, rather than silently excluded from the breach list
            if not entered or entered <= _iso_days_ago_from(int(sd), now):
                sla.append({"deal_id": d["id"], "stage": d["stage"],
                            "since": entered or "unknown", "sla_days": sd})
        hot = [a for a in self.a.read_log("activities") if a.get("type") == "email_reply"][-20:]
        return {"generated_at": now,
                "tasks_due": due_tasks, "open_tasks": len(tasks),
                "deals_open": len(deals), "sla_breaches": sla,
                "hot_replies": [{"lead_id": a.get("contact_id"), "at": a.get("ts")} for a in hot],
                "drafts_pending": len(self.list_pending_drafts())}

    def _has_open_task(self, contact_id: str, title: str) -> bool:
        rc = self.resolve(contact_id)
        return any(t.get("status") == "open" and t.get("title") == title and t.get("contact_id") == rc
                   for t in self._latest_tasks())

    def _latest_tasks(self) -> list:
        latest = {}
        for t in self.a.read_log("tasks"):
            latest[t["id"]] = t  # last write per id wins
        return list(latest.values())

    def render_today_view(self, now: str | None = None) -> dict:
        d = self.today_view_data(now)
        md = [f"# Today — {d['generated_at'][:16].replace('T',' ')}",
              f"**{d['drafts_pending']}** drafts awaiting approval  ·  **{len(d['tasks_due'])}** tasks due  ·  "
              f"**{len(d['sla_breaches'])}** deals past SLA  ·  **{d['deals_open']}** open deals", ""]
        md += ["## Tasks due", ""]
        md += [f"- {t.get('title','')}  (due {t.get('due_at','')[:16]})" for t in d["tasks_due"]] or ["*(none)*"]
        md += ["", "## Deals past SLA", ""]
        md += [f"- deal {b['deal_id'][:10]} stuck at `{b['stage']}` since {b['since'][:10]} (SLA {b['sla_days']}d)"
               for b in d["sla_breaches"]] or ["*(none)*"]
        md += ["", "## Hot replies (respond fast)", ""]
        md += [f"- reply from {r['lead_id']} at {r['at'][:16]}" for r in d["hot_replies"]] or ["*(none)*"]
        return self._render_operator(md, "today-view", "Today View", now)

    def render_kanban(self, now: str | None = None) -> dict:
        now = now or now_iso()
        pipelines = self.get_pipelines().get("pipelines", [])
        deals = self.a.query("deals", where=[Cond("status", "=", "open")])
        md = [f"# Pipeline — {now[:10]}", ""]
        forecast = 0.0
        for p in pipelines:
            for st in p.get("stages", []):
                if st["id"] in ("won", "lost"):
                    continue
                col = [d for d in deals if d.get("stage") == st["id"]]
                md.append(f"## {st['id']} ({len(col)})")
                for d in col:
                    v = _as_float(d.get("value")); prob = _as_float(d.get("probability"))
                    forecast += v * prob
                    md.append(f"- {d.get('name') or d['id'][:10]} — ${v:.0f} × {prob:.0%}")
                md.append("")
        md.insert(1, f"**Weighted forecast:** ${forecast:,.0f}  ·  {len(deals)} open deals\n")
        return self._render_operator(md, "kanban", "Pipeline Kanban", now)

    # --- weekly client report (CLIENT-FACING, scrubbed, Mondays) -------------

    _STAGE_LABELS = {"new_reply": "New replies", "engaged": "In conversation",
                     "meeting_booked": "Meeting booked", "proposal_sent": "Proposal sent",
                     "won": "Won", "lost": "Closed out"}

    def _contact_display(self, lead_id: str) -> str:
        c = self.get_contact(self.resolve(lead_id)) if lead_id else None
        if not c:
            return "A prospect"
        nm = (c.get("name") or {}).get("full") or ""
        return nm.strip() or "A prospect"

    def weekly_report_data(self, now: str | None = None, days: int = 7) -> dict:
        """Aggregate the last `days` of CRM state into client-facing figures. Pure counts and
        names — never internal identifiers, campaign slugs, or provider/tooling terms (the scrub
        gate is the backstop, but the source is built clean)."""
        now = now or now_iso()
        # Floor the cutoff to midnight UTC so the displayed date-only period_start is truthfully
        # inclusive of the whole day (activity earlier the same day is not silently dropped).
        cutoff = _iso_days_ago_from(days, now)[:10] + "T00:00:00Z"
        acts = [a for a in self.a.read_log("activities") if (a.get("ts") or "") >= cutoff]
        delivered = sum(1 for a in acts if a.get("type") == "email_sent")
        replies = sum(1 for a in acts if a.get("type") == "email_reply")
        deals = self.a.query("deals", where=[Cond("status", "=", "open")])
        forecast = 0.0
        by_stage = {}
        for d in deals:
            v = _as_float(d.get("value")); prob = _as_float(d.get("probability"))
            forecast += v * prob
            st = d.get("stage", "")
            agg = by_stage.setdefault(st, {"count": 0, "value": 0.0})
            agg["count"] += 1; agg["value"] += v
        # movements: any stage entered within the window (across open + closed deals)
        movements = []
        for d in self.a.query("deals", where=[]):
            name = self._contact_display((d.get("contact_ids") or [None])[0])
            for h in d.get("stage_history", []):
                if (h.get("at") or "") >= cutoff:
                    movements.append({"name": name, "stage": h.get("stage", ""),
                                      "at": h.get("at", ""), "value": _as_float(d.get("value")),
                                      "status": d.get("status", "open")})
        movements.sort(key=lambda m: m["at"])
        meetings = sum(1 for m in movements if m["stage"] == "meeting_booked")
        won = [m for m in movements if m["stage"] == "won"]
        # next steps: open deals in the two closing stages
        next_steps = [{"name": self._contact_display((d.get("contact_ids") or [None])[0]),
                       "stage": d.get("stage", "")}
                      for d in deals if d.get("stage") in ("meeting_booked", "proposal_sent")]
        return {"generated_at": now, "period_start": cutoff[:10], "period_end": now[:10], "days": days,
                "delivered": delivered, "replies": replies,
                # cap at 100%: replies this week may answer sends from before the window
                "reply_rate": min(replies / delivered, 1.0) if delivered else None,
                "new_conversations": sum(1 for m in movements if m["stage"] == "new_reply"),
                "meetings": meetings, "open_deals": len(deals), "forecast": forecast,
                "by_stage": by_stage, "movements": movements, "won": won, "next_steps": next_steps}

    def render_weekly_report(self, now: str | None = None, client_name: str = "", days: int = 7) -> dict:
        """Build + render the client-facing weekly report (scrub-gated). Returns the render
        status incl. `blocked` + `blind_terms` if the scrub gate fired."""
        d = self.weekly_report_data(now, days)
        cname = client_name.strip() or self._client_slug().replace("-", " ").replace("_", " ").title()
        rr = "—" if d["reply_rate"] is None else f"{d['reply_rate']:.0%}"
        md = [f"# {cname} — Weekly Outreach Report",
              f"### {d['period_start']} to {d['period_end']}", "",
              f"**This week:** {d['delivered']} emails delivered · {d['replies']} replies · "
              f"{d['new_conversations']} new conversations · ${d['forecast']:,.0f} in active pipeline", "",
              "## Snapshot", "",
              f"- Emails delivered: **{d['delivered']}**",
              f"- Replies received: **{d['replies']}** ({rr})",
              f"- New conversations started: **{d['new_conversations']}**",
              f"- Meetings booked: **{d['meetings']}**",
              f"- Active opportunities: **{d['open_deals']}**", "",
              "## Pipeline", "",
              f"**Weighted pipeline value: ${d['forecast']:,.0f}** across {d['open_deals']} opportunities", "",
              "| Stage | Opportunities | Value |", "|---|---:|---:|"]
        for st in ("new_reply", "engaged", "meeting_booked", "proposal_sent"):
            agg = d["by_stage"].get(st)
            if agg:
                md.append(f"| {self._STAGE_LABELS[st]} | {agg['count']} | ${agg['value']:,.0f} |")
        md += ["", "## What moved this week", ""]
        moved = []
        for m in d["movements"]:
            label = self._STAGE_LABELS.get(m["stage"], m["stage"])
            if m["stage"] == "won":
                moved.append(f"- **Won** — {m['name']} (${m['value']:,.0f})")
            elif m["stage"] == "lost":
                moved.append(f"- Closed out — {m['name']}")
            else:
                moved.append(f"- {m['name']} → {label}")
        md += moved or ["Conversations are progressing; no stage changes to report this week."]
        md += ["", "## What's next", ""]
        md += [f"- {n['name']} — {self._STAGE_LABELS.get(n['stage'], n['stage']).lower()} in progress"
               for n in d["next_steps"]] or ["We continue outreach and follow-ups on the active list."]
        md += ["", "---", f"Prepared by your outreach team · {d['period_end']}"]
        return self._render_client_facing(md, "weekly-client-report", f"{cname} — Weekly Report",
                                          cname, "Weekly Client Report", now)

    def _render_operator(self, md_lines: list, slug_suffix: str, title: str, now: str | None) -> dict:
        now = now or now_iso()
        out_dir = os.path.join(self.client_dir, "outputs", today_str(now))
        os.makedirs(out_dir, exist_ok=True)
        md_path = os.path.join(out_dir, f"{self._client_slug()}-{slug_suffix}.md")
        html_path = os.path.join(out_dir, f"{self._client_slug()}-{slug_suffix}.html")
        JsonAdapter._atomic_write(md_path, "\n".join(md_lines))
        rendered = False
        try:
            import subprocess
            r = subprocess.run(["python3", os.path.join(os.path.dirname(os.path.abspath(__file__)), "report_renderer.py"),
                                "render", "--input", md_path, "--output-html", html_path, "--title", title,
                                "--report-date", now[:10]],  # honor the (possibly injected) clock
                               capture_output=True, text=True, timeout=60)
            rendered = r.returncode == 0
        except Exception:  # noqa: BLE001
            rendered = False
        return {"md": md_path, "html": html_path if rendered else None, "html_rendered": rendered}

    def _render_client_facing(self, md_lines: list, slug_suffix: str, title: str,
                              client_name: str, report_kind: str, now: str | None) -> dict:
        """Render through the Client-Blind Scrub Gate. On a blind-term hit the renderer exits 3
        and writes only a `.blocked.html` sidecar — we surface `blocked: True` + the offending
        terms and never present the real path, so a contaminated report is never shipped."""
        now = now or now_iso()
        out_dir = os.path.join(self.client_dir, "outputs", today_str(now))
        os.makedirs(out_dir, exist_ok=True)
        md_path = os.path.join(out_dir, f"{self._client_slug()}-{slug_suffix}.md")
        html_path = os.path.join(out_dir, f"{self._client_slug()}-{slug_suffix}.html")
        JsonAdapter._atomic_write(md_path, "\n".join(md_lines))
        try:
            import subprocess
            r = subprocess.run(["python3", os.path.join(os.path.dirname(os.path.abspath(__file__)), "report_renderer.py"),
                                "render", "--input", md_path, "--output-html", html_path,
                                "--title", title, "--client-name", client_name,
                                "--report-kind", report_kind, "--report-date", now[:10],
                                "--client-facing", "--fail-on-scrub"],
                               capture_output=True, text=True, timeout=60)
        except Exception as e:  # noqa: BLE001
            return {"md": md_path, "html": None, "html_rendered": False, "blocked": False, "error": str(e)}
        if r.returncode == 3:
            blind = []
            try:
                blind = json.loads(r.stderr).get("client_blind_terms_found", [])
            except Exception:  # noqa: BLE001
                pass
            return {"md": md_path, "html": None, "html_rendered": False, "blocked": True, "blind_terms": blind}
        rendered = r.returncode in (0, 2)  # 2 = html clean but PDF engine unavailable
        return {"md": md_path, "html": html_path if rendered else None,
                "html_rendered": rendered, "blocked": False}

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
            title = args.get("title", "Follow up")
            # don't create a second identical OPEN task for the same real contact (e.g. the same
            # rule fired under a pre-merge alias id) — mirror create_deal_if_none's resolved guard
            if cid and self._has_open_task(cid, title):
                return None
            due = args.get("due", "")
            return {"task_id": self.add_task(title, contact_id=cid,
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

def _as_float(value, default: float = 0.0) -> float:
    """Coerce to float, never raising: JSON null, missing, or a non-numeric string
    (e.g. an LLM-produced confidence of 'high') all fall back to `default`."""
    try:
        if value is None:
            return default
        return float(value)
    except (TypeError, ValueError):
        return default


_EVIDENCE_URL_RE = re.compile(r"^https?://[^\s/]+", re.I)  # scheme + a host — rejects 'N/A', bare words, blanks


def _valid_evidence_url(url) -> bool:
    """A hook's evidence_url must be a real http(s) URL with a host — not 'N/A',
    whitespace, or a bare word — so a fabricated 'source' can't reach an outbound email."""
    return isinstance(url, str) and bool(_EVIDENCE_URL_RE.match(url.strip()))


_EMAIL_RE = re.compile(r"^[^@\s]+@[^@\s]+\.[^@\s]+$")


def _valid_email(address) -> bool:
    """A real, well-formed address — so a garbage string ('not-an-email') can't pass the
    email_first 'has a usable email' guard and burn a sendbox slot on a guaranteed bounce."""
    return isinstance(address, str) and bool(_EMAIL_RE.match(address.strip()))


def _subject_gate_normalized(subject: str) -> str:
    """NFKC-fold + strip Unicode format/control chars so the step-1 Re:/Fwd: anti-deception
    gate can't be bypassed with a zero-width space or a fullwidth colon."""
    norm = unicodedata.normalize("NFKC", subject or "")
    return "".join(ch for ch in norm if unicodedata.category(ch) != "Cf")


def _safe_slug(slug: str, what: str = "slug") -> str:
    """Reject path-traversal / separator tricks in a user-supplied slug (mirrors
    JsonAdapter._safe_id) before it is used to build a filesystem path."""
    if not slug or "/" in slug or "\\" in slug or slug in (".", "..") or slug.startswith("."):
        raise StorageError(f"unsafe {what} {slug!r}")
    return slug


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
        base = _dt.datetime.fromisoformat(now_iso().replace("Z", "+00:00"))
        return (base + delta).isoformat().replace("+00:00", "Z")
    return due


def _iso_days_ago(days: int) -> str:
    return _iso_days_ago_from(days, now_iso())


def _iso_days_ago_from(days: int, now_iso_str: str) -> str:
    """The ISO timestamp `days` before the given now (honors an injected clock)."""
    import datetime as _dt
    base = _dt.datetime.fromisoformat(now_iso_str.replace("Z", "+00:00"))
    return (base - _dt.timedelta(days=days)).isoformat().replace("+00:00", "Z")


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
    en = sub.add_parser("enrich"); en.add_argument("op", choices=["status", "due", "write", "get"])
    en.add_argument("--contact"); en.add_argument("--campaign"); en.add_argument("--json"); en.add_argument("--limit", type=int, default=100)
    dr = sub.add_parser("draft"); dr.add_argument("op", choices=["write", "list"])
    dr.add_argument("--contact"); dr.add_argument("--campaign"); dr.add_argument("--json")
    aprt = sub.add_parser("approval-report"); aprt.add_argument("--campaign")
    apy = sub.add_parser("approve"); apy.add_argument("--json", required=True)
    fu = sub.add_parser("followups"); fu.add_argument("op", choices=["due"]); fu.add_argument("--campaign", required=True)
    tv = sub.add_parser("today-view")
    kb = sub.add_parser("kanban")
    wr = sub.add_parser("weekly-report"); wr.add_argument("--client-name", default=""); wr.add_argument("--days", type=int, default=7)

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
    if args.cmd == "enrich":
        if args.op == "status":
            return _out(store.enrich_status(args.contact))
        if args.op == "due":
            return _out(store.enrich_due(args.campaign, args.limit))
        if args.op == "write":
            return _out(store.enrich_write(args.contact, json.loads(args.json), campaign_slug=args.campaign))
        if args.op == "get":
            c = store.get_contact(args.contact) or {}
            return _out(c.get("enrichment") or {})
    if args.cmd == "draft":
        if args.op == "write":
            d = json.loads(args.json)
            return _out(store.draft_write(args.contact, args.campaign, d.get("step", 1),
                                          d.get("subject", ""), d.get("body_text", ""),
                                          hooks_used=d.get("hooks_used"), body_html=d.get("body_html", ""),
                                          tracking=d.get("tracking", "plain_text")))
        if args.op == "list":
            return _out(store.list_pending_drafts(args.campaign))
    if args.cmd == "approval-report":
        return _out(store.render_approval_report(args.campaign))
    if args.cmd == "approve":
        return _out(store.approve_apply(json.loads(args.json)))
    if args.cmd == "followups":
        return _out(store.followups_due(args.campaign))
    if args.cmd == "today-view":
        return _out(store.render_today_view())
    if args.cmd == "kanban":
        return _out(store.render_kanban())
    if args.cmd == "weekly-report":
        return _out(store.render_weekly_report(client_name=args.client_name, days=args.days))
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
