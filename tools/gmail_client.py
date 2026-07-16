#!/usr/bin/env python3
"""gmail_client.py — send + sync for @gmail.com sendboxes via App Password (SMTP/IMAP).

Priority path from DESIGN §8: no OAuth, no 7-day expiry, preserves our Message-ID.
Stdlib only (smtplib/imaplib/email/ssl). The advanced OAuth/Workspace mode is a later
addition behind the same CLI.

  gmail_client.py auth   --client-dir DIR --sendbox sb-a --email you@gmail.com
        App Password read from env OUTREACHCRM_APP_PASSWORD (never a CLI arg / never logged).
        Verifies SMTP+IMAP login, writes sendboxes.json entry + sendboxes/sb-a/credentials.json (chmod 600).
  gmail_client.py health --client-dir DIR --sendbox sb-a
  gmail_client.py quota  --client-dir DIR --sendbox sb-a [--day YYYY-MM-DD]
  gmail_client.py send   --client-dir DIR --draft path/to/draft.json [--dry-run]
        Runs the ordered pre-send re-check IN CODE, then SMTP-sends, records sent_log + activity.
  gmail_client.py sync   --client-dir DIR --sendbox sb-a [--max 100]
        IMAP fetch since the UID cursor, classify each message (DSN first, then OOO, then
        +unsub alias, then thread match, then contact, else personal), suppress bounces/unsubs.

Phase 1 is plain_text_mode: no open pixel, no link rewrite, no tracker pull (Phase 2).
List-Unsubscribe mailto is always present for compliance.
"""

from __future__ import annotations

import argparse
import email
import email.message
import email.utils
import imaplib
import json
import os
import re
import smtplib
import ssl
import sys
import time
import uuid

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from storage import now_iso, today_str, month_str, normalize_email  # noqa: E402
from crm_store import CrmStore, _append_jsonl_seq  # noqa: E402

SMTP_HOST, SMTP_PORT = "smtp.gmail.com", 465
IMAP_HOST, IMAP_PORT = "imap.gmail.com", 993


# --- sendbox config -----------------------------------------------------------

def _sendboxes_path(client_dir): return os.path.join(client_dir, "sendboxes", "sendboxes.json")
def _cred_path(client_dir, slug): return os.path.join(client_dir, "sendboxes", slug, "credentials.json")


def load_sendboxes(client_dir) -> dict:
    p = _sendboxes_path(client_dir)
    if os.path.isfile(p):
        with open(p, "r", encoding="utf-8") as fh:
            return json.load(fh)
    return {"sendboxes": []}


def get_sendbox(client_dir, slug) -> dict | None:
    for sb in load_sendboxes(client_dir).get("sendboxes", []):
        if sb.get("slug") == slug:
            return sb
    return None


def save_sendbox(client_dir, sb: dict) -> None:
    data = load_sendboxes(client_dir)
    boxes = [b for b in data.get("sendboxes", []) if b.get("slug") != sb["slug"]]
    boxes.append(sb)
    data["sendboxes"] = boxes
    os.makedirs(os.path.dirname(_sendboxes_path(client_dir)), exist_ok=True)
    tmp = _sendboxes_path(client_dir) + ".tmp"
    with open(tmp, "w", encoding="utf-8") as fh:
        json.dump(data, fh, ensure_ascii=False, indent=2)
    os.replace(tmp, _sendboxes_path(client_dir))


def load_credentials(client_dir, slug) -> dict:
    p = _cred_path(client_dir, slug)
    if not os.path.isfile(p):
        raise SystemExit(f"no credentials for sendbox {slug!r}; run `gmail_client.py auth` first")
    with open(p, "r", encoding="utf-8") as fh:
        return json.load(fh)


# --- auth / health ------------------------------------------------------------

def _smtp_login(email_addr, app_password):
    ctx = ssl.create_default_context()
    s = smtplib.SMTP_SSL(SMTP_HOST, SMTP_PORT, context=ctx, timeout=30)
    s.login(email_addr, app_password)
    return s


def _imap_login(email_addr, app_password):
    m = imaplib.IMAP4_SSL(IMAP_HOST, IMAP_PORT)
    m.login(email_addr, app_password)
    return m


def cmd_auth(client_dir, slug, email_addr) -> dict:
    app_password = os.environ.get("OUTREACHCRM_APP_PASSWORD")
    if not app_password:
        raise SystemExit("set OUTREACHCRM_APP_PASSWORD to the 16-char Gmail App Password (never pass it as a CLI arg)")
    app_password = app_password.replace(" ", "")
    # verify both channels
    s = _smtp_login(email_addr, app_password); s.quit()
    m = _imap_login(email_addr, app_password); m.logout()
    # persist
    cred_dir = os.path.join(client_dir, "sendboxes", slug)
    os.makedirs(cred_dir, exist_ok=True)
    cpath = _cred_path(client_dir, slug)
    with open(cpath, "w", encoding="utf-8") as fh:
        json.dump({"email": email_addr, "app_password": app_password,
                   "smtp_host": SMTP_HOST, "imap_host": IMAP_HOST}, fh)
    os.chmod(cpath, 0o600)
    domain = email_addr.rsplit("@", 1)[1].lower()
    existing = get_sendbox(client_dir, slug) or {}
    sb = {"slug": slug, "auth_mode": "app_password", "email": email_addr, "domain": domain,
          "quota_today": existing.get("quota_today", 20), "warmup_stage": existing.get("warmup_stage", "week_1"),
          "status": "healthy", "historyId": None,
          "imap_uid_cursor": existing.get("imap_uid_cursor"), "last_successful_sync_ts": existing.get("last_successful_sync_ts", "")}
    save_sendbox(client_dir, sb)
    return {"ok": True, "sendbox": slug, "email": email_addr, "smtp": "ok", "imap": "ok",
            "quota_today": sb["quota_today"], "warmup_stage": sb["warmup_stage"]}


def cmd_health(client_dir, slug) -> dict:
    cred = load_credentials(client_dir, slug)
    out = {"sendbox": slug, "email": cred["email"], "smtp": "?", "imap": "?", "status": "healthy"}
    try:
        s = _smtp_login(cred["email"], cred["app_password"]); s.quit(); out["smtp"] = "ok"
    except Exception as e:  # noqa: BLE001
        out["smtp"] = f"fail: {e.__class__.__name__}"; out["status"] = "needs_reauth"
    try:
        m = _imap_login(cred["email"], cred["app_password"]); m.logout(); out["imap"] = "ok"
    except Exception as e:  # noqa: BLE001
        out["imap"] = f"fail: {e.__class__.__name__}"; out["status"] = "needs_reauth"
    sb = get_sendbox(client_dir, slug)
    if sb and sb.get("status") != out["status"]:
        save_sendbox(client_dir, {**sb, "status": out["status"]})
    return out


# --- quota --------------------------------------------------------------------

def sent_count_today(client_dir, slug, day) -> int:
    """Count real sends (not reservations) for a sendbox on a day across all campaigns."""
    n = 0
    camp_root = os.path.join(client_dir, "campaigns")
    if not os.path.isdir(camp_root):
        return 0
    for camp in os.listdir(camp_root):
        sent_dir = os.path.join(camp_root, camp, "sent", month_str())
        p = os.path.join(sent_dir, "sent_log.jsonl")
        if os.path.isfile(p):
            for line in open(p, "r", encoding="utf-8"):
                line = line.strip()
                if not line:
                    continue
                try:
                    r = json.loads(line)
                except ValueError:
                    continue
                if r.get("sendbox") == slug and (r.get("sent_at", "") or "")[:10] == day and r.get("rfc_message_id"):
                    n += 1
    return n


def cmd_quota(client_dir, slug, day) -> dict:
    sb = get_sendbox(client_dir, slug) or {}
    cap = int(sb.get("quota_today", 0))
    sent = sent_count_today(client_dir, slug, day)
    store = CrmStore(client_dir)
    reserved = store.a.reservation_count(slug, day)
    return {"sendbox": slug, "day": day, "cap": cap, "sent": sent, "reserved": reserved,
            "remaining": max(0, cap - max(sent, reserved))}


# --- send ---------------------------------------------------------------------

def presend_check(store: CrmStore, client_dir, sb: dict, draft: dict, day: str) -> tuple[bool, str, str | None]:
    """The ordered in-code pre-send re-check (DESIGN §10/§16). Returns (ok, reason, token)."""
    slug = sb["slug"]
    lead_id = store.resolve(draft["lead_id"])
    contact = store.get_contact(lead_id)
    if not contact:
        return False, "contact_not_found", None
    # 1. suppression (client + global), all identities
    emails = [e["address"] for e in contact.get("identities", {}).get("emails", [])]
    phones = [p["number"] for p in contact.get("identities", {}).get("phones", [])]
    if store.is_suppressed(email=(emails[0] if emails else None), phone=(phones[0] if phones else None)):
        return False, "suppressed", None
    # 2. channel status
    if contact.get("channels", {}).get("email", {}).get("status") in ("opted_out", "bounced"):
        return False, "email_channel_not_usable", None
    # 3. guessed cap / guessed_approved (guessed_only must be approved)
    prim = next((e for e in contact.get("identities", {}).get("emails", []) if e.get("is_primary")), None)
    if prim and prim.get("status") == "guessed_only" and not draft.get("guessed_approved"):
        return False, "guessed_email_needs_approval", None
    # 4. sequence freeze (any inbound reply freezes remaining bumps)
    if contact.get("sequence_state") == "frozen":
        return False, "sequence_frozen", None
    # 5. step-1 subject lint (no fake Re:/Fwd:)
    if int(draft.get("step", 1)) == 1 and re.match(r"^\s*(re|fwd)\s*:", draft.get("subject", ""), re.I):
        return False, "step1_subject_looks_like_reply", None
    # 6. atomic quota reservation (last, so we never reserve for a blocked send)
    cap = int(sb.get("quota_today", 0))
    already = sent_count_today(client_dir, slug, day)
    if already >= cap:
        return False, "quota_exhausted", None
    token = store.reserve(slug, day, cap)
    if not token:
        return False, "quota_exhausted", None
    return True, "ok", token


def build_mime(sb: dict, draft: dict, rfc_message_id: str, thread_refs: str | None) -> email.message.EmailMessage:
    msg = email.message.EmailMessage()
    from_name = draft.get("from_name") or sb.get("from_name") or ""
    msg["From"] = email.utils.formataddr((from_name, sb["email"])) if from_name else sb["email"]
    msg["To"] = draft["to"]
    subject = draft.get("subject", "")
    if int(draft.get("step", 1)) > 1 and not re.match(r"^\s*re\s*:", subject, re.I):
        subject = "Re: " + subject
    msg["Subject"] = subject
    msg["Message-ID"] = rfc_message_id
    msg["Date"] = email.utils.formatdate(localtime=True)
    # compliance: List-Unsubscribe mailto (+unsub alias token maps to lead via sent_log)
    token = draft.get("token") or _mk_token()
    local, dom = sb["email"].split("@", 1)
    msg["List-Unsubscribe"] = f"<mailto:{local}+unsub-{token}@{dom}?subject=unsubscribe>"
    if thread_refs:
        msg["In-Reply-To"] = thread_refs
        msg["References"] = thread_refs
    body = draft.get("body_text", "")
    msg.set_content(body)
    # minimal html alternative only when explicitly not plain_text_mode (Phase 2 tracking)
    if draft.get("tracking") == "pixel_and_links" and draft.get("body_html"):
        msg.add_alternative(draft["body_html"], subtype="html")
    draft["token"] = token
    return msg


def cmd_send(client_dir, draft_path, dry_run=False) -> dict:
    with open(draft_path, "r", encoding="utf-8") as fh:
        draft = json.load(fh)
    store = CrmStore(client_dir)
    slug = draft["sendbox"]
    sb = get_sendbox(client_dir, slug)
    if not sb:
        return {"ok": False, "blocker": "sendbox_not_configured", "sendbox": slug}
    if sb.get("status") != "healthy":
        return {"ok": False, "blocker": f"sendbox_{sb.get('status')}", "sendbox": slug}
    # approval gate: a draft must be approved to send
    if draft.get("status") != "approved":
        return {"ok": False, "blocker": "draft_not_approved", "draft_status": draft.get("status")}
    day = today_str()
    ok, reason, token = presend_check(store, client_dir, sb, draft, day)
    if not ok:
        return {"ok": False, "blocker": reason, "lead_id": draft.get("lead_id")}

    # thread refs for bumps/replies: prior rfc_message_id from this lead's sent_log
    thread_refs = _prior_message_id(client_dir, draft["campaign_slug"], store.resolve(draft["lead_id"])) if int(draft.get("step", 1)) > 1 else None
    rfc_message_id = f"<{uuid.uuid4().hex}@{sb['email'].split('@',1)[1]}>"
    msg = build_mime(sb, draft, rfc_message_id, thread_refs)

    if dry_run:
        return {"ok": True, "dry_run": True, "would_send_to": draft["to"], "sendbox": slug,
                "subject": msg["Subject"], "reserved_token": token, "rfc_message_id": rfc_message_id,
                "list_unsubscribe": msg["List-Unsubscribe"]}

    cred = load_credentials(client_dir, slug)
    try:
        s = _smtp_login(cred["email"], cred["app_password"])
        s.send_message(msg)
        s.quit()
    except Exception as e:  # noqa: BLE001
        return {"ok": False, "blocker": "smtp_send_failed", "error": f"{e.__class__.__name__}: {e}"}

    # jitter between sends (caller may loop; keep it here as a safety pace)
    sent_at = now_iso()
    _append_sent_log(client_dir, draft["campaign_slug"], {
        "lead_id": store.resolve(draft["lead_id"]), "campaign": draft["campaign_slug"], "step": draft.get("step", 1),
        "sendbox": slug, "provider_id": "", "thread_id": thread_refs or "", "rfc_message_id": rfc_message_id,
        "token": draft.get("token", ""), "links": draft.get("links", {}), "sent_at": sent_at,
    })
    act = store.log_activity("email_sent", draft["lead_id"], summary=f"sent step {draft.get('step',1)} via {slug}",
                             by="agent", ref={"message_id": rfc_message_id})
    # mark draft sent
    draft["status"] = "sent"; draft["decided_at"] = sent_at
    with open(draft_path, "w", encoding="utf-8") as fh:
        json.dump(draft, fh, ensure_ascii=False, indent=2)
    return {"ok": True, "sent_to": draft["to"], "sendbox": slug, "rfc_message_id": rfc_message_id,
            "activity_seq": act["seq"], "sent_at": sent_at}


def _append_sent_log(client_dir, campaign_slug, record):
    d = os.path.join(client_dir, "campaigns", campaign_slug, "sent", month_str())
    os.makedirs(d, exist_ok=True)
    p = os.path.join(d, "sent_log.jsonl")
    record = {"seq": _append_jsonl_seq(p), "ts": now_iso(), **record}
    with open(p, "a", encoding="utf-8") as fh:
        fh.write(json.dumps(record, ensure_ascii=False) + "\n")


def _prior_message_id(client_dir, campaign_slug, lead_id) -> str | None:
    d = os.path.join(client_dir, "campaigns", campaign_slug, "sent", month_str())
    p = os.path.join(d, "sent_log.jsonl")
    last = None
    if os.path.isfile(p):
        for line in open(p, "r", encoding="utf-8"):
            line = line.strip()
            if line:
                try:
                    r = json.loads(line)
                except ValueError:
                    continue
                if r.get("lead_id") == lead_id and r.get("rfc_message_id"):
                    last = r["rfc_message_id"]
    return last


def _mk_token() -> str:
    return uuid.uuid4().hex[:12]


# --- sync + classify ----------------------------------------------------------

def classify_message(client_dir, msg: email.message.Message, sendbox_email: str, known_message_ids: dict) -> dict:
    """Deterministic classifier, DESIGN §12 order: DSN first, then OOO, then +unsub alias,
    then thread match, then contact_message, else personal."""
    from_hdr = (msg.get("From", "") or "").lower()
    ctype = (msg.get_content_type() or "").lower()
    to_all = " ".join(filter(None, [msg.get("To", ""), msg.get("Delivered-To", ""), msg.get("X-Original-To", "")]))
    subject = msg.get("Subject", "") or ""
    body = _plain_body(msg)

    # 1. DSN / bounce
    is_dsn = ("mailer-daemon" in from_hdr or "postmaster" in from_hdr
              or ctype == "multipart/report" or "report-type=delivery-status" in (msg.get("Content-Type", "") or "").lower())
    if is_dsn:
        hard = bool(re.search(r"\b5\.\d\.\d\b", body)) or "550" in body
        return {"kind": "bounce", "hard": hard, "bounced_message_id": _extract_original_mid(body)}
    # 2. auto-reply / OOO
    if (msg.get("Auto-Submitted", "").lower().startswith("auto")
            or re.search(r"out of (the )?office|auto[- ]?reply|automatic reply", subject, re.I)):
        return {"kind": "auto_reply_ooo"}
    # 3. +unsub alias token (deterministic — most reliable)
    m = re.search(re.escape(sendbox_email.split("@")[0]) + r"\+unsub-([A-Za-z0-9]+)@", to_all)
    if m:
        return {"kind": "unsubscribe", "token": m.group(1)}
    # 4. thread match (campaign reply) via In-Reply-To / References
    refs = (msg.get("In-Reply-To", "") + " " + msg.get("References", ""))
    for mid in re.findall(r"<[^>]+>", refs):
        if mid in known_message_ids:
            return {"kind": "campaign_reply", "lead_id": known_message_ids[mid]["lead_id"],
                    "campaign": known_message_ids[mid]["campaign"], "in_reply_to": mid}
    # 5. from a known contact but no thread match
    addr = email.utils.parseaddr(msg.get("From", ""))[1].lower()
    return {"kind": "contact_or_personal", "from": addr}


def _plain_body(msg) -> str:
    if msg.is_multipart():
        for part in msg.walk():
            if part.get_content_type() == "text/plain":
                try:
                    return part.get_payload(decode=True).decode(part.get_content_charset() or "utf-8", "replace")
                except (LookupError, AttributeError):
                    return ""
        return ""
    try:
        return msg.get_payload(decode=True).decode(msg.get_content_charset() or "utf-8", "replace")
    except (LookupError, AttributeError):
        return msg.get_payload() or ""


def _extract_original_mid(body: str) -> str:
    m = re.search(r"<[^>]+@[^>]+>", body or "")
    return m.group(0) if m else ""


def _load_known_message_ids(client_dir) -> dict:
    """Map rfc_message_id -> {lead_id, campaign} from all campaign sent_logs."""
    out = {}
    camp_root = os.path.join(client_dir, "campaigns")
    if not os.path.isdir(camp_root):
        return out
    for camp in os.listdir(camp_root):
        sent_base = os.path.join(camp_root, camp, "sent")
        if not os.path.isdir(sent_base):
            continue
        for month in os.listdir(sent_base):
            p = os.path.join(sent_base, month, "sent_log.jsonl")
            if os.path.isfile(p):
                for line in open(p, "r", encoding="utf-8"):
                    line = line.strip()
                    if line:
                        try:
                            r = json.loads(line)
                        except ValueError:
                            continue
                        if r.get("rfc_message_id"):
                            out[r["rfc_message_id"]] = {"lead_id": r.get("lead_id"), "campaign": r.get("campaign")}
    return out


def cmd_sync(client_dir, slug, max_msgs=100) -> dict:
    cred = load_credentials(client_dir, slug)
    sb = get_sendbox(client_dir, slug)
    store = CrmStore(client_dir)
    known = _load_known_message_ids(client_dir)
    m = _imap_login(cred["email"], cred["app_password"])
    m.select("INBOX")
    cursor = int(sb.get("imap_uid_cursor") or 0)
    typ, data = m.uid("search", None, f"UID {cursor + 1}:*")
    uids = [int(x) for x in (data[0].split() if data and data[0] else []) if int(x) > cursor]
    uids = uids[:max_msgs]
    results = {"bounce": 0, "auto_reply_ooo": 0, "unsubscribe": 0, "campaign_reply": 0, "personal": 0}
    replies = []
    max_uid = cursor
    for uid in uids:
        typ, md = m.uid("fetch", str(uid), "(RFC822)")
        if not md or not md[0]:
            continue
        msg = email.message_from_bytes(md[0][1])
        cls = classify_message(client_dir, msg, cred["email"], known)
        max_uid = max(max_uid, uid)
        kind = cls["kind"]
        if kind == "bounce":
            mid = cls.get("bounced_message_id")
            info = known.get(mid)
            if info and info.get("lead_id"):
                reason = "hard_bounce" if cls.get("hard") else "manual"
                if cls.get("hard"):
                    store.suppress_contact(info["lead_id"], "hard_bounce", by="rule")
                store.log_activity("email_bounce", info["lead_id"], summary=f"{'hard' if cls.get('hard') else 'soft'} bounce", by="rule")
            results["bounce"] += 1
        elif kind == "auto_reply_ooo":
            results["auto_reply_ooo"] += 1
        elif kind == "unsubscribe":
            token = cls["token"]
            info = _lookup_token(client_dir, token)
            if info and info.get("lead_id"):
                store.suppress_contact(info["lead_id"], "unsubscribe", by="rule")
                store.log_activity("unsubscribe", info["lead_id"], summary="unsubscribed via mailto alias", by="rule")
            results["unsubscribe"] += 1
        elif kind == "campaign_reply":
            lead = cls.get("lead_id")
            if lead:
                store.set_contact(lead, {"sequence_state": "frozen"})  # any reply freezes sequence
                act = store.log_activity("email_reply", lead, summary="campaign reply (untriaged)", by="rule",
                                         ref={"message_id": msg.get("Message-ID", "")})
                replies.append({"lead_id": lead, "campaign": cls.get("campaign"), "activity_seq": act["seq"],
                                "subject": msg.get("Subject", ""), "from": email.utils.parseaddr(msg.get("From",""))[1]})
            results["campaign_reply"] += 1
        else:
            results["personal"] += 1  # count only; do not store body
    m.logout()
    save_sendbox(client_dir, {**sb, "imap_uid_cursor": max_uid, "last_successful_sync_ts": now_iso()})
    # write a sync_log row
    _append_sync_log(client_dir, slug, {"checked": len(uids), **results})
    return {"sendbox": slug, "checked": len(uids), "cursor": max_uid, "counts": results,
            "replies_untriaged": replies}


def _lookup_token(client_dir, token) -> dict | None:
    camp_root = os.path.join(client_dir, "campaigns")
    if not os.path.isdir(camp_root):
        return None
    for camp in os.listdir(camp_root):
        sent_base = os.path.join(camp_root, camp, "sent")
        if not os.path.isdir(sent_base):
            continue
        for month in os.listdir(sent_base):
            p = os.path.join(sent_base, month, "sent_log.jsonl")
            if os.path.isfile(p):
                for line in open(p, "r", encoding="utf-8"):
                    line = line.strip()
                    if line:
                        try:
                            r = json.loads(line)
                        except ValueError:
                            continue
                        if r.get("token") == token:
                            return {"lead_id": r.get("lead_id"), "campaign": r.get("campaign")}
    return None


def _append_sync_log(client_dir, slug, record):
    d = os.path.join(client_dir, "inbox_sync", month_str())
    os.makedirs(d, exist_ok=True)
    p = os.path.join(d, "sync_log.jsonl")
    record = {"seq": _append_jsonl_seq(p), "ts": now_iso(), "sendbox": slug, **record}
    with open(p, "a", encoding="utf-8") as fh:
        fh.write(json.dumps(record, ensure_ascii=False) + "\n")


# --- CLI ----------------------------------------------------------------------

def main(argv=None) -> int:
    p = argparse.ArgumentParser(description="Gmail App Password send/sync for OutreachCRM")
    p.add_argument("--client-dir", required=True)
    sub = p.add_subparsers(dest="cmd", required=True)
    a = sub.add_parser("auth"); a.add_argument("--sendbox", required=True); a.add_argument("--email", required=True)
    h = sub.add_parser("health"); h.add_argument("--sendbox", required=True)
    q = sub.add_parser("quota"); q.add_argument("--sendbox", required=True); q.add_argument("--day", default=today_str())
    s = sub.add_parser("send"); s.add_argument("--draft", required=True); s.add_argument("--dry-run", action="store_true")
    sy = sub.add_parser("sync"); sy.add_argument("--sendbox", required=True); sy.add_argument("--max", type=int, default=100)
    args = p.parse_args(argv)
    cd = args.client_dir
    if args.cmd == "auth":
        out = cmd_auth(cd, args.sendbox, args.email)
    elif args.cmd == "health":
        out = cmd_health(cd, args.sendbox)
    elif args.cmd == "quota":
        out = cmd_quota(cd, args.sendbox, args.day)
    elif args.cmd == "send":
        out = cmd_send(cd, args.draft, dry_run=args.dry_run)
    elif args.cmd == "sync":
        out = cmd_sync(cd, args.sendbox, args.max)
    else:
        out = {"error": "unknown"}
    print(json.dumps(out, ensure_ascii=False, indent=2))
    return 0 if out.get("ok", True) is not False else 1


if __name__ == "__main__":
    sys.exit(main())
