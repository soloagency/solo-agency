"""OutreachCRM pluggable storage.

`get_adapter(client_root)` returns the backend named in
`outreach-pipeline/storage_config.json` (default: json). All CRM mutations go
through `tools/crm_store.py`, which is the only sanctioned writer; this package is
the storage mechanism it drives. Stdlib-only so scheduled agents need no install.
"""

from .adapter import (
    Cond, StorageError, get_adapter, new_ulid, now_iso, today_str, month_str,
    normalize_email, normalize_phone, normalize_social,
)

__all__ = [
    "Cond",
    "StorageError",
    "get_adapter",
    "new_ulid",
    "now_iso",
    "today_str",
    "month_str",
    "normalize_email",
    "normalize_phone",
    "normalize_social",
]
