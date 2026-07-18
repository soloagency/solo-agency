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
import html as _html
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
    # verify both channels, and baseline the IMAP cursor to the mailbox's current top UID
    # so the first sync only ever sees mail that arrives AFTER this connection (incl. replies
    # to our sends) rather than flooding on the whole mailbox history.
    s = _smtp_login(email_addr, app_password); s.quit()
    m = _imap_login(email_addr, app_password)
    baseline = None
    try:
        m.select("INBOX")
        typ, data = m.uid("search", None, "ALL")
        all_uids = [int(x) for x in (data[0].split() if data and data[0] else [])]
        baseline = max(all_uids) if all_uids else 0
    finally:
        m.logout()
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
    # keep an existing cursor on re-auth; otherwise seed the baseline from this connection
    cursor = existing.get("imap_uid_cursor")
    if cursor is None:
        cursor = baseline
    sb = {"slug": slug, "auth_mode": "app_password", "email": email_addr, "domain": domain,
          "quota_today": existing.get("quota_today", 20), "warmup_stage": existing.get("warmup_stage", "week_1"),
          "status": "healthy", "historyId": None,
          "imap_uid_cursor": cursor, "last_successful_sync_ts": existing.get("last_successful_sync_ts", "")}
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
        for p in _sent_log_files(client_dir, camp):  # all months (a day near a month boundary)
            with open(p, "r", encoding="utf-8") as fh:
                _lines = fh.readlines()
            for line in _lines:
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

def _header_safe(value: str) -> bool:
    return "\n" not in (value or "") and "\r" not in (value or "")


def _identity_path(client_dir):
    return os.path.join(client_dir, "config", "sending_identity.json")


def load_sending_identity(client_dir) -> dict:
    """Machine-readable sending identity (config/sending_identity.json), written at Stage 1
    alongside the human Client Intelligence Profile. Plain config like sendboxes.json."""
    p = _identity_path(client_dir)
    if not os.path.isfile(p):
        return {}
    try:
        with open(p, "r", encoding="utf-8") as fh:
            return json.load(fh) or {}
    except ValueError:
        return {}


def compliance_footer(identity: dict) -> str | None:
    """CAN-SPAM footer appended to EVERY outgoing body (physical postal address + visible
    opt-out line; 15 U.S.C. 7704). Returns None when no physical address is configured so
    presend_check can fail closed instead of sending a non-compliant email."""
    addr = (identity.get("physical_mailing_address") or "").strip()
    if not addr:
        return None
    name = (identity.get("from_name") or "").strip()
    optout = (identity.get("unsubscribe_text") or "").strip() or \
        'Don\'t want these emails? Reply "unsubscribe" and we\'ll stop.'
    lines = ["-- "]
    if name:
        lines.append(name)
    lines.append(addr)
    lines.append(optout)
    return "\n".join(lines)


def presend_check(store: CrmStore, client_dir, sb: dict, draft: dict, day: str,
                  reserve: bool = True) -> tuple[bool, str, str | None]:
    """The ordered in-code pre-send re-check (DESIGN §10/§16). Returns (ok, reason, token).
    Checks suppression against EVERY identity AND the actual recipient. Set reserve=False
    for a dry-run so a preview never consumes quota."""
    slug = sb["slug"]
    lead_id = store.resolve(draft["lead_id"])
    contact = store.get_contact(lead_id)
    if not contact:
        return False, "contact_not_found", None
    ids = contact.get("identities", {})
    emails = [e.get("address") for e in ids.get("emails", []) if e.get("address")]
    phones = [p.get("number") for p in ids.get("phones", []) if p.get("number")]
    socials = [v for v in (ids.get("socials", {}) or {}).values() if v]
    to_addr = (draft.get("to") or "").strip()
    # 0. the actual recipient must be one of the resolved contact's own emails (no mis-send)
    if not to_addr or normalize_email(to_addr) not in {normalize_email(e) for e in emails}:
        return False, "recipient_not_a_contact_identity", None
    if not (_header_safe(draft.get("subject", "")) and _header_safe(to_addr)):
        return False, "invalid_draft_headers", None
    # 1. suppression (client + global) across ALL identities AND the recipient address
    for addr in set(emails) | {to_addr}:
        if store.is_suppressed(email=addr):
            return False, "suppressed", None
    for ph in phones:
        if store.is_suppressed(phone=ph):
            return False, "suppressed", None
    if socials and store.is_suppressed(socials=socials):
        return False, "suppressed", None
    # 2. channel status
    if contact.get("channels", {}).get("email", {}).get("status") in ("opted_out", "bounced"):
        return False, "email_channel_not_usable", None
    # 2b. CAN-SPAM: a sending identity with a physical mailing address must exist
    # (config/sending_identity.json) — fail closed; the body footer is legally required
    if compliance_footer(load_sending_identity(client_dir)) is None:
        return False, "missing_physical_address", None
    # 3. guessed_only must be approved
    prim = next((e for e in ids.get("emails", []) if e.get("is_primary")), None)
    if prim and prim.get("status") == "guessed_only" and not draft.get("guessed_approved"):
        return False, "guessed_email_needs_approval", None
    # 4. sequence freeze (any inbound reply freezes remaining bumps)
    if contact.get("sequence_state") == "frozen":
        return False, "sequence_frozen", None
    # 5. step-1 subject lint (no fake Re:/Fwd:)
    if int(draft.get("step", 1)) == 1 and re.match(r"^\s*(re|fwd)\s*:", draft.get("subject", ""), re.I):
        return False, "step1_subject_looks_like_reply", None
    # 6. double-send guard: already sent this (lead, campaign, step)?
    if _already_sent(client_dir, draft["campaign_slug"], lead_id, int(draft.get("step", 1))):
        return False, "already_sent", None
    # 7. sticky sender: step>1 must use the box that sent step-1
    assigned = contact.get("assigned_sendbox")
    if int(draft.get("step", 1)) > 1 and assigned and assigned != slug:
        return False, "wrong_sendbox_for_sticky_sender", None
    # 8. atomic quota reservation (last; skipped on dry-run so previews don't burn quota)
    cap = int(sb.get("quota_today", 0))
    if sent_count_today(client_dir, slug, day) >= cap:
        return False, "quota_exhausted", None
    if not reserve:
        return True, "ok", None
    token = store.reserve(slug, day, cap)
    if not token:
        return False, "quota_exhausted", None
    return True, "ok", token


def _already_sent(client_dir, campaign_slug, lead_id, step) -> bool:
    for camp_month in _sent_log_files(client_dir, campaign_slug):
        with open(camp_month, "r", encoding="utf-8") as fh:
            _lines = fh.readlines()
        for line in _lines:
            line = line.strip()
            if not line:
                continue
            try:
                r = json.loads(line)
            except ValueError:
                continue
            if r.get("lead_id") == lead_id and int(r.get("step", 0)) == int(step) and r.get("rfc_message_id"):
                return True
    return False


def _sent_log_files(client_dir, campaign_slug):
    base = os.path.join(client_dir, "campaigns", campaign_slug, "sent")
    if not os.path.isdir(base):
        return []
    out = []
    for month in sorted(os.listdir(base)):
        p = os.path.join(base, month, "sent_log.jsonl")
        if os.path.isfile(p):
            out.append(p)
    return out


def build_mime(sb: dict, draft: dict, rfc_message_id: str, thread_refs: str | None,
               footer: str | None = None) -> email.message.EmailMessage:
    msg = email.message.EmailMessage()
    from_name = draft.get("from_name") or sb.get("from_name") or ""
    msg["From"] = email.utils.formataddr((from_name, sb["email"])) if from_name else sb["email"]
    msg["To"] = draft["to"]
    subject = draft.get("subject", "")
    # only claim "Re:" when this is a genuine in-thread reply (we have the prior msg-id)
    if int(draft.get("step", 1)) > 1 and thread_refs and not re.match(r"^\s*re\s*:", subject, re.I):
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
    # CAN-SPAM footer (physical address + opt-out) rides on EVERY send, incl. plain_text_mode
    if footer:
        body = body.rstrip() + "\n\n" + footer
    msg.set_content(body)
    # minimal html alternative only when explicitly not plain_text_mode (Phase 2 tracking)
    if draft.get("tracking") == "pixel_and_links" and draft.get("body_html"):
        html_body = draft["body_html"]
        if footer:
            html_body = html_body + "<br><br>" + _html.escape(footer).replace("\n", "<br>")
        msg.add_alternative(html_body, subtype="html")
    draft["token"] = token
    return msg


_TERMINAL_BLOCKERS = {"suppressed", "email_channel_not_usable", "sequence_frozen",
                      "recipient_not_a_contact_identity", "contact_not_found",
                      "already_sent", "invalid_draft_headers", "step1_subject_looks_like_reply"}


def _persist_send_blocker(draft_path, draft, reason):
    """A failed send must never be silent (Stage 9: 'never silently dropped'). Records the
    blocker on the draft record: TERMINAL blockers flip status to 'blocked' (do not retry);
    transient ones (quota, SMTP, auth, sendbox health) keep status 'approved' so the next
    run retries naturally, with blocker/blocked_at showing why the last attempt failed."""
    if draft.get("status") != "approved":
        return  # only an approved draft that failed to send gets annotated
    draft["blocker"] = reason
    draft["blocked_at"] = now_iso()
    if reason in _TERMINAL_BLOCKERS:
        draft["status"] = "blocked"
    with open(draft_path, "w", encoding="utf-8") as fh:
        json.dump(draft, fh, ensure_ascii=False, indent=2)


def cmd_send(client_dir, draft_path, dry_run=False) -> dict:
    with open(draft_path, "r", encoding="utf-8") as fh:
        draft = json.load(fh)
    store = CrmStore(client_dir)
    slug = draft["sendbox"]
    sb = get_sendbox(client_dir, slug)
    if not sb:
        if not dry_run:
            _persist_send_blocker(draft_path, draft, "sendbox_not_configured")
        return {"ok": False, "blocker": "sendbox_not_configured", "sendbox": slug}
    if sb.get("status") != "healthy":
        if not dry_run:
            _persist_send_blocker(draft_path, draft, f"sendbox_{sb.get('status')}")
        return {"ok": False, "blocker": f"sendbox_{sb.get('status')}", "sendbox": slug}
    # approval gate: a draft must be approved to send
    if draft.get("status") != "approved":
        return {"ok": False, "blocker": "draft_not_approved", "draft_status": draft.get("status")}
    day = today_str()
    lead_id = store.resolve(draft["lead_id"])
    step = int(draft.get("step", 1))
    ok, reason, token = presend_check(store, client_dir, sb, draft, day, reserve=not dry_run)
    if not ok:
        if not dry_run:
            _persist_send_blocker(draft_path, draft, reason)
        return {"ok": False, "blocker": reason, "lead_id": draft.get("lead_id")}

    # thread refs for bumps/replies: prior rfc_message_id from this lead's sent_log (all months)
    thread_refs = _prior_message_id(client_dir, draft["campaign_slug"], lead_id) if step > 1 else None
    rfc_message_id = f"<{uuid.uuid4().hex}@{sb['email'].split('@',1)[1]}>"
    footer = compliance_footer(load_sending_identity(client_dir))  # non-None (presend gate 2b)
    try:
        msg = build_mime(sb, draft, rfc_message_id, thread_refs, footer=footer)
    except (ValueError, TypeError) as e:  # header injection / malformed draft
        if token:
            store.a.release(slug, day, token)
        if not dry_run:
            _persist_send_blocker(draft_path, draft, "invalid_draft_headers")
        return {"ok": False, "blocker": "invalid_draft_headers", "error": str(e)}

    if dry_run:
        return {"ok": True, "dry_run": True, "would_send_to": draft["to"], "sendbox": slug,
                "subject": msg["Subject"], "rfc_message_id": rfc_message_id,
                "list_unsubscribe": msg["List-Unsubscribe"], "note": "dry-run: no quota reserved, nothing sent"}

    cred = load_credentials(client_dir, slug)
    try:
        s = _smtp_login(cred["email"], cred["app_password"])
        s.send_message(msg)
        s.quit()
    except Exception as e:  # noqa: BLE001 - do NOT surface e in a way that could echo the password
        if token:
            store.a.release(slug, day, token)  # send didn't happen; don't leak quota
        needs_reauth = "auth" in e.__class__.__name__.lower() or isinstance(e, smtplib.SMTPAuthenticationError)
        if needs_reauth:
            save_sendbox(client_dir, {**sb, "status": "needs_reauth"})
        _persist_send_blocker(draft_path, draft, "needs_reauth" if needs_reauth else "smtp_send_failed")
        return {"ok": False, "blocker": "needs_reauth" if needs_reauth else "smtp_send_failed",
                "error": e.__class__.__name__}

    # Record the send FIRST (sent_log is the durable idempotency source), then flip status.
    sent_at = now_iso()
    _append_sent_log(client_dir, draft["campaign_slug"], {
        "lead_id": lead_id, "campaign": draft["campaign_slug"], "step": step,
        "sendbox": slug, "provider_id": "", "thread_id": thread_refs or "", "rfc_message_id": rfc_message_id,
        "token": draft.get("token", ""), "links": draft.get("links", {}), "sent_at": sent_at,
    })
    act = store.log_activity("email_sent", lead_id, summary=f"sent step {step} via {slug}",
                             by="agent", ref={"message_id": rfc_message_id})
    # sticky sender: pin the sendbox on the first send so bumps reuse it
    patch = {}
    if step == 1 and not store.get_contact(lead_id).get("assigned_sendbox"):
        patch["assigned_sendbox"] = slug
    if patch:
        store.set_contact(lead_id, patch)
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
    last = None
    for p in _sent_log_files(client_dir, campaign_slug):  # all months, chronological
        with open(p, "r", encoding="utf-8") as fh:
            _lines = fh.readlines()
        for line in _lines:
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

def classify_message(client_dir, msg: email.message.Message, sendbox_email: str, known_message_ids: dict,
                     from_resolver=None) -> dict:
    """Deterministic classifier, DESIGN §12 order: DSN first, then OOO, then +unsub alias,
    then thread match, then known-contact (From-address fallback), else personal.
    from_resolver(email) -> {lead_id, campaign?} | None resolves a reply whose client did
    not echo In-Reply-To (common) so we still detect it and freeze the sequence."""
    from_hdr = (msg.get("From", "") or "").lower()
    ctype = (msg.get_content_type() or "").lower()
    to_all = " ".join(filter(None, [msg.get("To", ""), msg.get("Delivered-To", ""), msg.get("X-Original-To", "")]))
    subject = msg.get("Subject", "") or ""

    # 1. DSN / bounce — read the STRUCTURED parts, not the human prose
    is_dsn = ("mailer-daemon" in from_hdr or "postmaster" in from_hdr
              or ctype == "multipart/report" or "report-type=delivery-status" in (msg.get("Content-Type", "") or "").lower())
    if is_dsn:
        d = _dsn_details(msg)
        return {"kind": "bounce", "hard": d["hard"], "bounced_message_id": d["original_mid"],
                "final_recipient": d["final_recipient"]}
    # 2. auto-reply / OOO
    if (msg.get("Auto-Submitted", "").lower().startswith("auto")
            or re.search(r"out of (the )?office|auto[- ]?reply|automatic reply", subject, re.I)):
        return {"kind": "auto_reply_ooo"}
    # 3. +unsub alias token (deterministic — most reliable). Only the sendbox's OWN
    #    +unsub alias counts, and only in a To/Delivered-To/X-Original-To header.
    m = re.search(re.escape(sendbox_email.split("@")[0]) + r"\+unsub-([A-Za-z0-9]+)@" + re.escape(sendbox_email.split("@")[1]), to_all, re.I)
    if m:
        return {"kind": "unsubscribe", "token": m.group(1)}
    # 4. thread match (campaign reply) via In-Reply-To / References
    refs = (msg.get("In-Reply-To", "") + " " + msg.get("References", ""))
    for mid in re.findall(r"<[^>]+>", refs):
        if mid in known_message_ids:
            return {"kind": "campaign_reply", "lead_id": known_message_ids[mid]["lead_id"],
                    "campaign": known_message_ids[mid]["campaign"], "in_reply_to": mid}
    # 5. From-address fallback: a reply from a KNOWN contact whose client dropped In-Reply-To
    addr = email.utils.parseaddr(msg.get("From", ""))[1].lower()
    if from_resolver:
        info = from_resolver(addr)
        if info and info.get("lead_id"):
            return {"kind": "campaign_reply", "lead_id": info["lead_id"], "campaign": info.get("campaign"),
                    "matched_by": "from_address"}
    # 6. unknown sender -> personal (count only)
    return {"kind": "contact_or_personal", "from": addr}


def _dsn_details(msg) -> dict:
    """Pull the original Message-ID + final recipient + hardness from a DSN's structured
    parts (message/delivery-status, message/rfc822, text/rfc822-headers) rather than the
    human-readable notice — real Gmail NDRs do not put our Message-ID in the prose."""
    original_mid, final_recipient, status = "", "", ""
    for part in msg.walk():
        ct = (part.get_content_type() or "").lower()
        if ct == "message/delivery-status":
            # email parses this into a list of Message blocks (per-message + per-recipient);
            # read the fields from their headers, falling back to a raw-text regex.
            payload = part.get_payload()
            blocks = payload if isinstance(payload, list) else []
            for blk in blocks:
                try:
                    fr = blk.get("Final-Recipient", "") or blk.get("Original-Recipient", "")
                    st = blk.get("Status", "")
                except Exception:  # noqa: BLE001
                    fr, st = "", ""
                if fr and not final_recipient:
                    mr = re.search(r"([^\s;]+@[^\s;]+)", fr)
                    if mr:
                        final_recipient = mr.group(1).strip().strip("<>").lower()
                if st and not status:
                    ms = re.search(r"([245]\.\d+\.\d+)", st)
                    if ms:
                        status = ms.group(1)
            if not (final_recipient and status):
                txt = "\n".join(b.as_string() for b in blocks) if blocks else _decoded_text(part)
                if not final_recipient:
                    mr = re.search(r"(?im)^\s*(?:Final-Recipient|Original-Recipient)\s*:\s*(?:rfc822;)?\s*([^\s;]+@[^\s;]+)", txt)
                    if mr:
                        final_recipient = mr.group(1).strip().strip("<>").lower()
                if not status:
                    ms = re.search(r"(?im)^\s*Status\s*:\s*([245]\.\d+\.\d+)", txt)
                    if ms:
                        status = ms.group(1)
        elif ct in ("message/rfc822", "text/rfc822-headers"):
            sub = part.get_payload(0) if part.is_multipart() else email.message_from_string(_decoded_text(part))
            try:
                mid = sub.get("Message-ID", "") if sub else ""
            except Exception:  # noqa: BLE001
                mid = ""
            if mid and not original_mid:
                original_mid = mid.strip()
    body = _plain_body(msg)
    if not original_mid:
        m = re.search(r"(?im)^\s*(?:X-Google-Original-Message-ID|Message-ID)\s*:\s*(<[^>]+>)", body)
        if m:
            original_mid = m.group(1)
    hard = status.startswith("5") if status else (bool(re.search(r"\b5\d\d[\s-]", body)) or bool(re.search(r"\b5\.\d+\.\d+\b", body)))
    return {"original_mid": original_mid, "final_recipient": final_recipient, "hard": hard, "status": status}


def _decoded_text(part) -> str:
    try:
        payload = part.get_payload(decode=True)
        if payload is None:
            return part.get_payload() if isinstance(part.get_payload(), str) else ""
        return payload.decode(part.get_content_charset() or "utf-8", "replace")
    except (LookupError, AttributeError, TypeError):
        return ""


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
                with open(p, "r", encoding="utf-8") as fh:
                    _lines = fh.readlines()
                for line in _lines:
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

    def from_resolver(addr):
        lead = store.a.find_by_identity("email", normalize_email(addr))
        return {"lead_id": lead} if lead else None

    m = _imap_login(cred["email"], cred["app_password"])
    m.select("INBOX")
    raw_cursor = sb.get("imap_uid_cursor")
    # First sync (never synced): baseline to the mailbox's current top UID and process
    # NOTHING, so we never flood/classify years of old, unrelated mail.
    if raw_cursor is None:
        typ, data = m.uid("search", None, "ALL")
        all_uids = [int(x) for x in (data[0].split() if data and data[0] else [])]
        baseline = max(all_uids) if all_uids else 0
        m.logout()
        save_sendbox(client_dir, {**sb, "imap_uid_cursor": baseline, "last_successful_sync_ts": now_iso()})
        return {"sendbox": slug, "checked": 0, "cursor": baseline, "baseline_set": True,
                "counts": {"bounce": 0, "auto_reply_ooo": 0, "unsubscribe": 0, "campaign_reply": 0, "personal": 0},
                "replies_untriaged": [], "note": "first sync: baselined to current mailbox top; only new mail is processed from here"}

    cursor = int(raw_cursor)
    typ, data = m.uid("search", None, f"UID {cursor + 1}:*")
    uids = sorted(int(x) for x in (data[0].split() if data and data[0] else []) if int(x) > cursor)
    batch = uids[:max_msgs]
    results = {"bounce": 0, "auto_reply_ooo": 0, "unsubscribe": 0, "campaign_reply": 0, "personal": 0}
    replies = []
    # Advance the cursor only across the CONTIGUOUS run of successfully-processed UIDs, so a
    # transient fetch miss can never bury an unread earlier message behind the cursor.
    new_cursor = cursor
    for uid in batch:
        typ, md = m.uid("fetch", str(uid), "(RFC822)")
        if not md or not md[0]:
            break  # stop advancing; this uid is retried next run
        msg = email.message_from_bytes(md[0][1])
        cls = classify_message(client_dir, msg, cred["email"], known, from_resolver=from_resolver)
        kind = cls["kind"]
        if kind == "bounce":
            lead = None
            info = known.get(cls.get("bounced_message_id"))
            if info:
                lead = info.get("lead_id")
            if not lead and cls.get("final_recipient"):  # fallback: match by recipient address
                lead = store.a.find_by_identity("email", normalize_email(cls["final_recipient"]))
            if lead:
                if cls.get("hard"):
                    store.suppress_contact(lead, "hard_bounce", by="rule")
                    store.set_contact(lead, {"sequence_state": "frozen"})  # dead address: stop drafting bumps too
                store.log_activity("email_bounce", lead, summary=f"{'hard' if cls.get('hard') else 'soft'} bounce", by="rule")
            results["bounce"] += 1
        elif kind == "auto_reply_ooo":
            results["auto_reply_ooo"] += 1
        elif kind == "unsubscribe":
            info = _lookup_token(client_dir, cls["token"])
            if info and info.get("lead_id"):
                store.suppress_contact(info["lead_id"], "unsubscribe", by="rule")
                store.set_contact(info["lead_id"], {"sequence_state": "frozen"})  # opted out: never draft another bump
                store.log_activity("unsubscribe", info["lead_id"], summary="unsubscribed via mailto alias", by="rule")
            results["unsubscribe"] += 1
        elif kind == "campaign_reply":
            lead = cls.get("lead_id")
            if lead:
                store.set_contact(lead, {"sequence_state": "frozen"})  # any reply freezes the sequence
                act = store.log_activity("email_reply", lead, summary="campaign reply (untriaged)", by="rule",
                                         ref={"message_id": msg.get("Message-ID", "")})
                replies.append({"lead_id": lead, "campaign": cls.get("campaign"), "activity_seq": act["seq"],
                                "subject": msg.get("Subject", ""), "from": email.utils.parseaddr(msg.get("From", ""))[1],
                                "matched_by": cls.get("matched_by", "thread")})
            results["campaign_reply"] += 1
        else:
            results["personal"] += 1  # count only; do not store the body
        new_cursor = uid  # this uid processed; safe to advance the contiguous cursor
    m.logout()
    save_sendbox(client_dir, {**sb, "imap_uid_cursor": new_cursor, "last_successful_sync_ts": now_iso()})
    processed = sum(results.values())
    _append_sync_log(client_dir, slug, {"checked": processed, "backlog_remaining": max(0, len(uids) - processed), **results})
    return {"sendbox": slug, "checked": processed, "cursor": new_cursor,
            "backlog_remaining": max(0, len(uids) - processed), "counts": results, "replies_untriaged": replies}


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
                with open(p, "r", encoding="utf-8") as fh:
                    _lines = fh.readlines()
                for line in _lines:
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
