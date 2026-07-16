#!/usr/bin/env python3
"""import_leads.py — turn a CSV/TXT/XLSX list into contacts, deduped and suppression-checked.

  import_leads.py inspect --file leads.csv
      -> prints the first rows + a proposed column mapping for the human to confirm.

  import_leads.py import --client-dir DIR --file leads.csv --list-slug al-realtors \
      [--mapping '{"email":"Email","full_name":"Full Name","company":"Office Name","phone":"Cell Phone","website":"Website"}']
      -> mints lead_id ULIDs via crm_store, dedupes (find_by_identity), checks suppression,
         writes lists/{slug}/{leads.jsonl,list_manifest.json,import_log.md}. Idempotent by
         (file content + mapping) hash: a second import of the same file is a no-op.

Email is NOT required (DESIGN §7.1): a row with only a name + phone/social still imports.
Stdlib only. XLSX is read with a minimal zip/XML reader (no openpyxl needed).
"""

from __future__ import annotations

import argparse
import csv
import hashlib
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from storage import now_iso, normalize_email, normalize_phone  # noqa: E402
from crm_store import CrmStore  # noqa: E402
import email_verify  # noqa: E402

# header synonyms -> canonical field
_SYNONYMS = {
    "email": ["email", "e-mail", "email address", "mail"],
    "full_name": ["full name", "name", "contact name", "fullname"],
    "first_name": ["first name", "first", "firstname"],
    "last_name": ["last name", "last", "lastname", "surname"],
    "company": ["company", "office name", "brokerage", "organization", "org", "business"],
    "phone": ["cell phone", "mobile", "cell", "phone", "phone number", "office phone"],
    "website": ["website", "url", "web", "site"],
    "city": ["city", "office city"],
    "state": ["state", "office state"],
    "facebook": ["facebook", "fb"],
    "linkedin": ["linkedin"],
    "instagram": ["instagram", "ig"],
}


def _rows_from_file(path: str) -> tuple[list[str], list[dict]]:
    ext = os.path.splitext(path)[1].lower()
    if ext in (".csv", ".txt", ".tsv"):
        with open(path, "r", encoding="utf-8-sig", newline="") as fh:
            sample = fh.read(4096)
            fh.seek(0)
            if ext == ".txt" and "," not in sample and "\t" not in sample:
                # one-value-per-line (emails). Header-less.
                headers = ["email"]
                rows = [{"email": ln.strip()} for ln in fh if ln.strip()]
                return headers, rows
            try:
                dialect = csv.Sniffer().sniff(sample, delimiters=",\t;")
            except csv.Error:
                dialect = csv.excel
            reader = csv.DictReader(fh, dialect=dialect)
            headers = reader.fieldnames or []
            rows = [dict(r) for r in reader]
            return headers, rows
    if ext == ".xlsx":
        return _rows_from_xlsx(path)
    raise SystemExit(f"unsupported file type {ext!r}; use csv/txt/xlsx")


def _rows_from_xlsx(path: str):
    """Minimal XLSX reader: first worksheet, first row = headers. Stdlib zip + XML."""
    import zipfile
    import xml.etree.ElementTree as ET
    ns = "{http://schemas.openxmlformats.org/spreadsheetml/2006/main}"
    with zipfile.ZipFile(path) as z:
        shared = []
        if "xl/sharedStrings.xml" in z.namelist():
            root = ET.fromstring(z.read("xl/sharedStrings.xml"))
            for si in root.findall(f"{ns}si"):
                shared.append("".join(t.text or "" for t in si.iter(f"{ns}t")))
        sheet_name = next((n for n in z.namelist() if n.startswith("xl/worksheets/sheet")), None)
        if not sheet_name:
            raise SystemExit("xlsx has no worksheet")
        root = ET.fromstring(z.read(sheet_name))
        grid = []
        for row in root.iter(f"{ns}row"):
            cells = {}
            for c in row.findall(f"{ns}c"):
                ref = c.get("r", "")
                col = "".join(ch for ch in ref if ch.isalpha())
                v = c.find(f"{ns}v")
                if v is None:
                    val = ""
                elif c.get("t") == "s":
                    val = shared[int(v.text)]
                else:
                    val = v.text or ""
                cells[col] = val
            grid.append(cells)
    if not grid:
        return [], []
    cols = sorted({k for row in grid for k in row}, key=lambda s: (len(s), s))
    headers = [grid[0].get(c, "") for c in cols]
    rows = []
    for row in grid[1:]:
        rows.append({headers[i]: row.get(cols[i], "") for i in range(len(cols)) if headers[i]})
    return headers, rows


def propose_mapping(headers: list[str]) -> dict:
    mapping = {}
    lower = {h.lower().strip(): h for h in headers if h}
    for field, syns in _SYNONYMS.items():
        for syn in syns:
            if syn in lower:
                mapping[field] = lower[syn]
                break
    return mapping


def _idempotency_key(path: str, mapping: dict) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as fh:
        h.update(fh.read())
    h.update(json.dumps(mapping, sort_keys=True).encode())
    return h.hexdigest()[:16]


def _normalize_row(raw: dict, mapping: dict) -> dict:
    def g(field):
        col = mapping.get(field)
        return (raw.get(col, "") if col else "").strip()
    full = g("full_name") or " ".join(x for x in [g("first_name"), g("last_name")] if x)
    socials = {}
    for s in ("facebook", "linkedin", "instagram"):
        if g(s):
            socials[s] = g(s)
    return {
        "full_name": full, "first_name": g("first_name"), "last_name": g("last_name"),
        "email": normalize_email(g("email")), "phone": normalize_phone(g("phone")),
        "company": g("company"), "website": g("website"),
        "city": g("city"), "state": g("state"), "socials": socials,
    }


def _to_contact_fields(norm: dict) -> dict:
    emails = [{"address": norm["email"], "source": "import", "status": "unverified", "is_primary": True}] if norm["email"] else []
    phones = [{"number": norm["phone"], "type": "cell", "source": "import"}] if norm["phone"] else []
    fields = {
        "name": {"full": norm["full_name"], "first": norm["first_name"], "last": norm["last_name"]},
        "identities": {"emails": emails, "phones": phones, "socials": norm["socials"], "website": norm["website"] or None},
        "custom_fields": {k: norm[k] for k in ("company", "city", "state") if norm[k]},
    }
    if norm["email"]:
        fields["channels"] = {"email": {"status": "usable"}}
    return fields


def do_import(client_dir: str, file: str, list_slug: str, mapping: dict | None,
              mx_check: bool = True) -> dict:
    headers, rows = _rows_from_file(file)
    if mapping is None:
        mapping = propose_mapping(headers)
    if not mapping:
        raise SystemExit("could not infer a column mapping; pass --mapping explicitly")
    store = CrmStore(client_dir)
    list_dir = os.path.join(client_dir, "lists", list_slug)
    os.makedirs(list_dir, exist_ok=True)
    leads_path = os.path.join(list_dir, "leads.jsonl")
    manifest_path = os.path.join(list_dir, "list_manifest.json")
    idem = _idempotency_key(file, mapping)

    # idempotency: same file+mapping already imported -> no-op
    if os.path.isfile(manifest_path):
        try:
            with open(manifest_path, "r", encoding="utf-8") as _mf:
                prev = json.load(_mf)
            if prev.get("idempotency_key") == idem:
                return {"skipped": True, "reason": "already imported (idempotency_key match)", "manifest": prev}
        except (OSError, ValueError):
            pass

    created = matched = suppressed = skipped = 0
    seq = 0
    with open(leads_path, "w", encoding="utf-8") as lf:
        for raw in rows:
            seq += 1
            norm = _normalize_row(raw, mapping)
            if not (norm["email"] or norm["phone"] or norm["socials"] or norm["full_name"]):
                skipped += 1
                lf.write(json.dumps({"seq": seq, "ts": now_iso(), "raw": raw, "normalized": norm,
                                     "outcome": "skipped_invalid", "lead_id": None, "reason": "no identity or name"}) + "\n")
                continue
            # suppression check against ALL identities
            supp = store.is_suppressed(email=norm["email"] or None, phone=norm["phone"] or None,
                                       socials=list(norm["socials"].values()))
            if supp:
                suppressed += 1
                lf.write(json.dumps({"seq": seq, "ts": now_iso(), "raw": raw, "normalized": norm,
                                     "outcome": "suppressed", "lead_id": None, "reason": supp.get("reason")}) + "\n")
                continue
            # optional MX check (marks email status; does not block import)
            if mx_check and norm["email"]:
                v = email_verify.check(norm["email"])
                if not v["mx_ok"]:
                    norm["_email_status"] = "email_not_found"
            fields = _to_contact_fields(norm)
            if norm["email"] and norm.get("_email_status"):
                fields["identities"]["emails"][0]["status"] = norm["_email_status"]
            lead_id, outcome = store.add_contact(fields)
            if outcome == "created":
                created += 1
                store.log_activity("imported", lead_id, summary=f"imported from {list_slug}", by="agent",
                                   ref={"path": f"lists/{list_slug}"})
            else:
                matched += 1
            lf.write(json.dumps({"seq": seq, "ts": now_iso(), "raw": raw, "normalized": norm,
                                 "outcome": outcome, "lead_id": lead_id, "reason": ""}) + "\n")

    manifest = {
        "schema_version": 1, "list_slug": list_slug, "source_file": os.path.abspath(file),
        "source_format": os.path.splitext(file)[1].lstrip("."), "imported_at": now_iso(),
        "idempotency_key": idem, "column_mapping": mapping, "row_count": len(rows),
        "contacts_created": created, "contacts_matched_existing": matched,
        "suppressed_at_import": suppressed, "rows_skipped": skipped, "notes": "",
    }
    with open(manifest_path, "w", encoding="utf-8") as fh:
        json.dump(manifest, fh, ensure_ascii=False, indent=2)
    _append_import_log(list_dir, manifest)
    return {"skipped": False, "manifest": manifest}


def _append_import_log(list_dir: str, m: dict) -> None:
    path = os.path.join(list_dir, "import_log.md")
    new = not os.path.isfile(path)
    with open(path, "a", encoding="utf-8") as fh:
        if new:
            fh.write("# Import Log\n\n| Date | Source | Rows | Created | Matched | Suppressed | Skipped | Blocker |\n|---|---|---|---|---|---|---|---|\n")
        fh.write(f"| {m['imported_at']} | {os.path.basename(m['source_file'])} | {m['row_count']} | "
                 f"{m['contacts_created']} | {m['contacts_matched_existing']} | {m['suppressed_at_import']} | "
                 f"{m['rows_skipped']} | {'—' if not m['rows_skipped'] else 'see leads.jsonl'} |\n")


def main(argv=None) -> int:
    p = argparse.ArgumentParser(description="Import a lead list into contacts (deduped, suppression-checked)")
    sub = p.add_subparsers(dest="cmd", required=True)
    ins = sub.add_parser("inspect"); ins.add_argument("--file", required=True); ins.add_argument("--rows", type=int, default=5)
    imp = sub.add_parser("import")
    imp.add_argument("--client-dir", required=True); imp.add_argument("--file", required=True)
    imp.add_argument("--list-slug", required=True); imp.add_argument("--mapping")
    imp.add_argument("--no-mx-check", action="store_true")
    args = p.parse_args(argv)

    if args.cmd == "inspect":
        headers, rows = _rows_from_file(args.file)
        out = {"headers": headers, "proposed_mapping": propose_mapping(headers),
               "sample_rows": rows[:args.rows], "total_rows": len(rows),
               "note": "Confirm/adjust the mapping, then run: import_leads.py import --mapping '<json>'"}
        print(json.dumps(out, ensure_ascii=False, indent=2))
        return 0
    if args.cmd == "import":
        mapping = json.loads(args.mapping) if args.mapping else None
        res = do_import(args.client_dir, args.file, args.list_slug, mapping, mx_check=not args.no_mx_check)
        print(json.dumps(res, ensure_ascii=False, indent=2))
        return 0
    return 2


if __name__ == "__main__":
    sys.exit(main())
