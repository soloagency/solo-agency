"""Storage adapter interface + shared helpers (ULID, timestamps, identity normalization).

The interface (DESIGN §6) is small and backend-neutral so a Postgres adapter can
later pass the same contract tests:

    get(collection, id) -> dict | None
    put(collection, id, record) -> None                 # atomic, bumps updated_at
    update(collection, id, mutate_fn) -> dict            # read-modify-write under the collection lock
    delete(collection, id) -> None                       # rarely used; prefer tombstones
    query(collection, where, sort=None, limit=None, offset=None) -> [dict]
    append(log, record) -> None                          # append-only, stamps ts + monotonic seq
    read_log(log, since_seq=None, where=None) -> [dict]  # ordered by seq
    find_by_identity(kind, normalized_value) -> id | None
    reserve(sendbox_slug, day, cap) -> token | None      # atomic quota reservation

`Cond = (field, op, value)`, op in {=, !=, <, >, contains, in}. Flat fields only;
identity lookups use find_by_identity over the maintained contact_identities index.
"""

from __future__ import annotations

import datetime as dt
import json
import os
import re
import secrets
import time
from typing import Any, Callable, NamedTuple, Optional


class StorageError(RuntimeError):
    pass


class Cond(NamedTuple):
    field: str
    op: str  # "=", "!=", "<", ">", "contains", "in"
    value: Any


_ALLOWED_OPS = {"=", "!=", "<", ">", "contains", "in"}


# --- timestamps ---------------------------------------------------------------

def now_iso() -> str:
    """UTC ISO-8601 with a trailing Z, second precision."""
    return dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def today_str(now: Optional[str] = None) -> str:
    """YYYY-MM-DD for the given ISO timestamp (or now)."""
    s = now or now_iso()
    return s[:10]


def month_str(now: Optional[str] = None) -> str:
    """YYYY-MM for the given ISO timestamp (or now)."""
    s = now or now_iso()
    return s[:7]


# --- ULID (Crockford base32, 48-bit time + 80-bit randomness) -----------------

_CROCKFORD = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"


def new_ulid(prefix: str = "") -> str:
    """A monotonic-enough, sortable id. Not RFC-strict but ULID-shaped (26 chars)."""
    ms = int(time.time() * 1000) & ((1 << 48) - 1)
    rand = int.from_bytes(secrets.token_bytes(10), "big")  # 80 bits
    value = (ms << 80) | rand
    chars = []
    for _ in range(26):
        chars.append(_CROCKFORD[value & 0x1F])
        value >>= 5
    return prefix + "".join(reversed(chars))


# --- identity normalization (must be stable so the reverse index is unique) ---

def normalize_email(address: str) -> str:
    return (address or "").strip().lower()


def normalize_phone(number: str) -> str:
    """Best-effort E.164-ish: keep leading +, strip non-digits. US 10-digit -> +1."""
    if not number:
        return ""
    s = number.strip()
    plus = s.startswith("+")
    digits = re.sub(r"\D", "", s)
    if not digits:
        return ""
    if plus:
        return "+" + digits
    if len(digits) == 10:
        return "+1" + digits
    if len(digits) == 11 and digits.startswith("1"):
        return "+" + digits
    return "+" + digits


def normalize_social(url: str) -> str:
    """Canonical social URL: lowercase host+path, strip scheme/query/trailing slash/www."""
    if not url:
        return ""
    s = url.strip().lower()
    s = re.sub(r"^https?://", "", s)
    s = re.sub(r"^www\.", "", s)
    s = s.split("?", 1)[0].split("#", 1)[0]
    return s.rstrip("/")


# --- base adapter -------------------------------------------------------------

class BaseAdapter:
    """Abstract; subclasses implement all methods. Records carry
    schema_version/id/created_at/updated_at; the adapter stamps created_at/updated_at."""

    backend = "base"

    def get(self, collection: str, id: str) -> Optional[dict]:
        raise NotImplementedError

    def put(self, collection: str, id: str, record: dict) -> None:
        raise NotImplementedError

    def update(self, collection: str, id: str, mutate_fn: Callable[[dict], dict]) -> dict:
        raise NotImplementedError

    def delete(self, collection: str, id: str) -> None:
        raise NotImplementedError

    def query(self, collection: str, where=None, sort=None, limit=None, offset=None) -> list:
        raise NotImplementedError

    def append(self, log: str, record: dict) -> dict:
        raise NotImplementedError

    def read_log(self, log: str, since_seq: Optional[int] = None, where=None) -> list:
        raise NotImplementedError

    def find_by_identity(self, kind: str, normalized_value: str) -> Optional[str]:
        raise NotImplementedError

    def reserve(self, sendbox_slug: str, day: str, cap: int) -> Optional[str]:
        raise NotImplementedError

    # shared: evaluate a Cond list against a record (flat fields, dotted paths allowed)
    @staticmethod
    def matches(record: dict, where) -> bool:
        for cond in where or []:
            field, op, value = cond
            if op not in _ALLOWED_OPS:
                raise StorageError(f"unsupported op {op!r}")
            actual = _dig(record, field)
            if op == "=" and not (actual == value):
                return False
            if op == "!=" and not (actual != value):
                return False
            if op == "<" and not (actual is not None and actual < value):
                return False
            if op == ">" and not (actual is not None and actual > value):
                return False
            if op == "contains" and not (actual is not None and value in actual):
                return False
            if op == "in" and not (actual in value):
                return False
        return True


def _dig(record: dict, dotted: str):
    cur = record
    for part in dotted.split("."):
        if isinstance(cur, dict) and part in cur:
            cur = cur[part]
        else:
            return None
    return cur


def get_adapter(client_root: str, backend: Optional[str] = None) -> BaseAdapter:
    """Return the adapter for a client's crm/ root. Backend from arg, else
    storage_config.json at the pipeline root, else 'json'."""
    if backend is None:
        backend = _read_backend(client_root)
    if backend == "json":
        from .json_adapter import JsonAdapter
        return JsonAdapter(client_root)
    if backend == "postgres":
        raise StorageError("postgres adapter is a later phase (DESIGN §21); set storage_config backend=json for now")
    raise StorageError(f"unknown storage backend {backend!r}")


def _read_backend(client_root: str) -> str:
    # storage_config.json lives at the pipeline root: outreach-pipeline/storage_config.json,
    # i.e. two levels above clients/{slug}/... — search upward for it.
    d = os.path.abspath(client_root)
    for _ in range(8):
        cfg = os.path.join(d, "storage_config.json")
        if os.path.isfile(cfg):
            try:
                with open(cfg, "r", encoding="utf-8") as fh:
                    return json.load(fh).get("backend", "json")
            except (OSError, ValueError):
                return "json"
        parent = os.path.dirname(d)
        if parent == d:
            break
        d = parent
    return "json"
