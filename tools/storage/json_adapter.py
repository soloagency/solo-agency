"""JSON storage backend (default).

Layout, rooted at a client dir `outreach-pipeline/clients/{slug}/`:
  crm/{collection}/{id}.json        record collections: contacts, accounts, deals
  crm/activities/{YYYY-MM}/activities.jsonl   monthly append log
  crm/tasks/tasks.jsonl             append log
  crm/contact_identities.jsonl      reverse index (kind,value -> contact_id, unique)
  crm/suppression.jsonl             client-tier suppression
  crm/.locks/*.lock                 fcntl lockfiles
  sendboxes/_reservations/{slug}/{day}.json   atomic quota reservations

Atomic writes = temp + os.replace. Monotonic seq = a per-log counter file bumped
under the log lock. Concurrency between the daily run and an interactive
approve-and-send session is covered by per-collection/per-log fcntl locks plus the
update(mutate_fn) read-modify-write primitive.
"""

from __future__ import annotations

import fcntl
import json
import os
from contextlib import contextmanager
from typing import Callable, Optional

from .adapter import BaseAdapter, StorageError, month_str, new_ulid, now_iso

# Logs with a non-default path. Any other log name maps to crm/{log}.jsonl
# (contact_identities, suppression, _rule_guards, ...). "activities" is monthly (special-cased).
_LOG_PATHS = {
    "tasks": "tasks/tasks.jsonl",
}
_RESERVED_COLLECTION_NAMES = {"activities", "tasks", "contact_identities", "suppression"}


class JsonAdapter(BaseAdapter):
    backend = "json"

    def __init__(self, client_root: str):
        self.client_root = os.path.abspath(client_root)
        self.crm_root = os.path.join(self.client_root, "crm")
        self._locks_dir = os.path.join(self.crm_root, ".locks")

    # --- paths ----------------------------------------------------------------

    def _collection_dir(self, collection: str) -> str:
        if collection in _RESERVED_COLLECTION_NAMES:
            raise StorageError(f"{collection!r} is a log, not a record collection")
        return os.path.join(self.crm_root, collection)

    def _record_path(self, collection: str, id: str) -> str:
        _safe_id(id)
        return os.path.join(self._collection_dir(collection), f"{id}.json")

    def _log_path(self, log: str, when: Optional[str] = None) -> str:
        if log == "activities":
            return os.path.join(self.crm_root, "activities", month_str(when), "activities.jsonl")
        if log in _LOG_PATHS:
            return os.path.join(self.crm_root, _LOG_PATHS[log])
        # generic flat log: crm/{log}.jsonl  (contact_identities, suppression, _rule_guards, ...)
        _safe_id(log)
        return os.path.join(self.crm_root, f"{log}.jsonl")

    # --- locking + atomic write ----------------------------------------------

    @contextmanager
    def _lock(self, name: str):
        os.makedirs(self._locks_dir, exist_ok=True)
        path = os.path.join(self._locks_dir, f"{name}.lock")
        fh = open(path, "w")
        try:
            fcntl.flock(fh, fcntl.LOCK_EX)
            yield
        finally:
            fcntl.flock(fh, fcntl.LOCK_UN)
            fh.close()

    @staticmethod
    def _atomic_write(path: str, text: str) -> None:
        os.makedirs(os.path.dirname(path), exist_ok=True)
        tmp = f"{path}.tmp.{os.getpid()}"
        with open(tmp, "w", encoding="utf-8") as fh:
            fh.write(text)
            fh.flush()
            os.fsync(fh.fileno())
        os.replace(tmp, path)

    def _next_seq(self, log: str) -> int:
        # caller holds the log lock
        seq_file = os.path.join(self.crm_root, ".seq", f"{log.replace('/', '_')}.seq")
        os.makedirs(os.path.dirname(seq_file), exist_ok=True)
        cur = 0
        if os.path.isfile(seq_file):
            try:
                with open(seq_file) as _fh:
                    cur = int(_fh.read().strip() or "0")
            except (OSError, ValueError):
                cur = 0
        nxt = cur + 1
        self._atomic_write(seq_file, str(nxt))
        return nxt

    # --- records --------------------------------------------------------------

    def get(self, collection: str, id: str) -> Optional[dict]:
        path = self._record_path(collection, id)
        if not os.path.isfile(path):
            return None
        with open(path, "r", encoding="utf-8") as fh:
            return json.load(fh)

    def put(self, collection: str, id: str, record: dict) -> None:
        record = dict(record)
        record.setdefault("id", id)
        if record["id"] != id:
            raise StorageError(f"record id {record['id']!r} != path id {id!r}")
        record.setdefault("schema_version", 1)
        record.setdefault("created_at", now_iso())
        record["updated_at"] = now_iso()
        with self._lock(f"col_{collection}"):
            self._atomic_write(self._record_path(collection, id), json.dumps(record, ensure_ascii=False, indent=2))

    def update(self, collection: str, id: str, mutate_fn: Callable[[dict], dict]) -> dict:
        with self._lock(f"col_{collection}"):
            path = self._record_path(collection, id)
            if not os.path.isfile(path):
                raise StorageError(f"{collection}/{id} not found for update")
            with open(path, "r", encoding="utf-8") as fh:
                record = json.load(fh)
            record = mutate_fn(record)
            record["updated_at"] = now_iso()
            self._atomic_write(path, json.dumps(record, ensure_ascii=False, indent=2))
            return record

    def delete(self, collection: str, id: str) -> None:
        with self._lock(f"col_{collection}"):
            path = self._record_path(collection, id)
            if os.path.isfile(path):
                os.remove(path)

    def query(self, collection: str, where=None, sort=None, limit=None, offset=None) -> list:
        d = self._collection_dir(collection)
        out = []
        if os.path.isdir(d):
            for name in sorted(os.listdir(d)):
                if not name.endswith(".json"):
                    continue
                try:
                    with open(os.path.join(d, name), "r", encoding="utf-8") as fh:
                        rec = json.load(fh)
                except (OSError, ValueError):
                    continue
                if self.matches(rec, where):
                    out.append(rec)
        if sort:
            field, desc = sort if isinstance(sort, tuple) else (sort, False)
            out.sort(key=lambda r: (r.get(field) is None, r.get(field)), reverse=bool(desc))
        if offset:
            out = out[offset:]
        if limit is not None:
            out = out[:limit]
        return out

    # --- append-only logs -----------------------------------------------------

    def append(self, log: str, record: dict) -> dict:
        with self._lock(f"log_{log}"):
            record = dict(record)
            record["seq"] = self._next_seq(log)
            record.setdefault("ts", now_iso())
            path = self._log_path(log, record.get("ts"))
            os.makedirs(os.path.dirname(path), exist_ok=True)
            with open(path, "a", encoding="utf-8") as fh:
                fh.write(json.dumps(record, ensure_ascii=False) + "\n")
                fh.flush()
                os.fsync(fh.fileno())
            return record

    def read_log(self, log: str, since_seq: Optional[int] = None, where=None) -> list:
        rows = []
        if log == "activities":
            base = os.path.join(self.crm_root, "activities")
            files = []
            if os.path.isdir(base):
                for month in sorted(os.listdir(base)):
                    p = os.path.join(base, month, "activities.jsonl")
                    if os.path.isfile(p):
                        files.append(p)
        else:
            p = self._log_path(log)
            files = [p] if os.path.isfile(p) else []
        for p in files:
            with open(p, "r", encoding="utf-8") as fh:
                for line in fh:
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        rec = json.loads(line)
                    except ValueError:
                        continue
                    if since_seq is not None and rec.get("seq", 0) <= since_seq:
                        continue
                    if self.matches(rec, where):
                        rows.append(rec)
        rows.sort(key=lambda r: r.get("seq", 0))
        return rows

    # --- identity reverse index ----------------------------------------------

    def register_identity(self, kind: str, normalized_value: str, contact_id: str) -> None:
        """Append a reverse-index row (used by crm_store on identity add). Uniqueness
        is enforced by the caller checking find_by_identity first, under the same lock."""
        if not normalized_value:
            return
        self.append("contact_identities", {
            "kind": kind, "value": normalized_value, "contact_id": contact_id, "removed": False,
        })

    def find_by_identity(self, kind: str, normalized_value: str) -> Optional[str]:
        if not normalized_value:
            return None
        found = None
        for rec in self.read_log("contact_identities"):
            if rec.get("kind") == kind and rec.get("value") == normalized_value:
                found = None if rec.get("removed") else rec.get("contact_id")
        return found

    # --- atomic quota reservation --------------------------------------------

    def reserve(self, sendbox_slug: str, day: str, cap: int) -> Optional[str]:
        """Increment a per-sendbox/day counter under lock; return a token, or None if
        the cap is reached. Voided reservations are not reclaimed (a send failure is
        rare and a slightly conservative count is safer than a race)."""
        _safe_id(sendbox_slug)
        path = os.path.join(self.client_root, "sendboxes", "_reservations", sendbox_slug, f"{day}.json")
        with self._lock(f"reserve_{sendbox_slug}_{day}"):
            state = {"count": 0, "tokens": []}
            if os.path.isfile(path):
                try:
                    with open(path, "r", encoding="utf-8") as fh:
                        state = json.load(fh)
                except (OSError, ValueError):
                    state = {"count": 0, "tokens": []}
            if state["count"] >= cap:
                return None
            token = new_ulid("rsv_")
            state["count"] += 1
            state["tokens"].append({"token": token, "at": now_iso()})
            self._atomic_write(path, json.dumps(state, ensure_ascii=False, indent=2))
            return token

    def reservation_count(self, sendbox_slug: str, day: str) -> int:
        path = os.path.join(self.client_root, "sendboxes", "_reservations", sendbox_slug, f"{day}.json")
        if os.path.isfile(path):
            try:
                with open(path, "r", encoding="utf-8") as fh:
                    return int(json.load(fh).get("count", 0))
            except (OSError, ValueError):
                return 0
        return 0


def _safe_id(id: str) -> None:
    if not id or "/" in id or "\\" in id or id in (".", "..") or id.startswith("."):
        raise StorageError(f"unsafe id {id!r}")
