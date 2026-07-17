"""JSON storage backend (default).

Layout, rooted at a client dir `daily-content-pipeline/clients/{slug}/{business}_{location}/outreach/`:
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

import json
import os
import threading
import uuid
from contextlib import contextmanager
from typing import Callable, Optional

try:
    import fcntl  # POSIX only
    _HAVE_FCNTL = True
except ImportError:  # pragma: no cover - Windows
    _HAVE_FCNTL = False

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
        if not _HAVE_FCNTL:
            raise StorageError("the json storage backend requires a POSIX (fcntl-capable) OS")
        self.client_root = os.path.abspath(client_root)
        self.crm_root = os.path.join(self.client_root, "crm")
        self._locks_dir = os.path.join(self.crm_root, ".locks")
        self._identity_cache = None  # {(kind,value): contact_id | None}; built lazily, kept in sync
        self._identity_cache_sig = None  # (size, mtime_ns) of the identity log when cache built

    # --- paths ----------------------------------------------------------------

    def _collection_dir(self, collection: str) -> str:
        if collection in _RESERVED_COLLECTION_NAMES:
            raise StorageError(f"{collection!r} is a log, not a record collection")
        _safe_id(collection)  # block path traversal into another client's tree
        d = os.path.join(self.crm_root, collection)
        root = os.path.realpath(self.crm_root)
        if os.path.commonpath([os.path.realpath(d), root]) != root:
            raise StorageError(f"collection {collection!r} escapes the client crm root")
        return d

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
        # unique per call (pid + thread + random) so two concurrent writers to the SAME path
        # never share a temp file and race on os.replace (a per-pid name is not enough).
        tmp = f"{path}.tmp.{os.getpid()}.{threading.get_ident()}.{uuid.uuid4().hex}"
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

    def _identity_log_sig(self):
        p = self._log_path("contact_identities")
        try:
            st = os.stat(p)
            return (st.st_size, st.st_mtime_ns)
        except OSError:
            return (0, 0)

    def _ensure_identity_cache(self) -> dict:
        """Build the (kind,value)->contact_id index once (turns per-lookup O(N) log
        scans / O(N^2) imports into O(1)), and rebuild only if ANOTHER process wrote to
        the log since (detected by a cheap size+mtime signature) — so a concurrent
        cross-process add can never dedupe against a stale in-memory cache."""
        sig = self._identity_log_sig()
        if self._identity_cache is None or sig != self._identity_cache_sig:
            cache = {}
            for rec in self.read_log("contact_identities"):
                cache[(rec.get("kind"), rec.get("value"))] = None if rec.get("removed") else rec.get("contact_id")
            self._identity_cache = cache
            self._identity_cache_sig = sig
        return self._identity_cache

    def register_identity(self, kind: str, normalized_value: str, contact_id: str) -> None:
        """Append a reverse-index row (used by crm_store on identity add). Uniqueness
        is enforced by the caller checking find_by_identity first, under the same lock."""
        if not normalized_value:
            return
        self.append("contact_identities", {
            "kind": kind, "value": normalized_value, "contact_id": contact_id, "removed": False,
        })
        if self._identity_cache is not None:
            self._identity_cache[(kind, normalized_value)] = contact_id
            self._identity_cache_sig = self._identity_log_sig()  # our own write; stay valid

    def find_by_identity(self, kind: str, normalized_value: str) -> Optional[str]:
        if not normalized_value:
            return None
        return self._ensure_identity_cache().get((kind, normalized_value))

    # --- atomic quota reservation --------------------------------------------

    def _reservation_path(self, sendbox_slug: str, day: str) -> str:
        _safe_id(sendbox_slug)
        return os.path.join(self.client_root, "sendboxes", "_reservations", sendbox_slug, f"{day}.json")

    @staticmethod
    def _load_reservation_state(path: str) -> dict:
        if os.path.isfile(path):
            try:
                with open(path, "r", encoding="utf-8") as fh:
                    st = json.load(fh)
                    if isinstance(st, dict):
                        st.setdefault("count", 0)
                        st.setdefault("tokens", [])
                        return st
            except (OSError, ValueError):
                pass
        return {"count": 0, "tokens": []}

    def reserve(self, sendbox_slug: str, day: str, cap: int) -> Optional[str]:
        """Increment a per-sendbox/day counter under lock; return a token, or None if
        the cap is reached. Callers MUST release() the token if the send does not
        actually happen (dry-run / SMTP failure) so previews and failures don't leak quota."""
        path = self._reservation_path(sendbox_slug, day)
        with self._lock(f"reserve_{sendbox_slug}_{day}"):
            state = self._load_reservation_state(path)
            if state["count"] >= cap:
                return None
            token = new_ulid("rsv_")
            state["count"] += 1
            state["tokens"].append({"token": token, "at": now_iso()})
            self._atomic_write(path, json.dumps(state, ensure_ascii=False, indent=2))
            return token

    def release(self, sendbox_slug: str, day: str, token: str) -> bool:
        """Return a reservation to the pool (send didn't happen). Idempotent."""
        path = self._reservation_path(sendbox_slug, day)
        with self._lock(f"reserve_{sendbox_slug}_{day}"):
            state = self._load_reservation_state(path)
            toks = [t for t in state["tokens"] if t.get("token") != token]
            if len(toks) == len(state["tokens"]):
                return False
            state["tokens"] = toks
            state["count"] = max(0, state["count"] - 1)
            self._atomic_write(path, json.dumps(state, ensure_ascii=False, indent=2))
            return True

    def reservation_count(self, sendbox_slug: str, day: str) -> int:
        return int(self._load_reservation_state(self._reservation_path(sendbox_slug, day)).get("count", 0))


def _safe_id(id: str) -> None:
    if not id or "/" in id or "\\" in id or id in (".", "..") or id.startswith("."):
        raise StorageError(f"unsafe id {id!r}")
