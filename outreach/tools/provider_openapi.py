#!/usr/bin/env python3
"""Minimal OpenAPI provider adapter for OutreachCRM (operator notification provider).

This utility intentionally uses only Python's standard library so scheduled
agents can run it without installing dependencies. It supports the subset
OutreachCRM needs: discover operation IDs, verify the account, send the operator
notification (e.g. WideCast sendTelegramMessage), and upload the HTML report link
through a provider operation such as WideCast uploadAsset. Production/video
operations are intentionally not part of OutreachCRM's provider role.
"""

from __future__ import annotations

import argparse
import base64
import json
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any


DEFAULT_DISCOVERY_URL = "https://widecast.ai/openapi.yaml"
DEFAULT_WIDECAST_SERVER_URL = "https://widecast.ai/app/dashboard"
DEFAULT_DISABLED_SERVER_URLS = {
    "widecast": {"https://api.widecast.ai"},
}
HTTP_METHODS = {"get", "post", "put", "patch", "delete"}
USER_AGENT = "OutreachCRMOpenAPIAdapter/1.0"

# OutreachCRM's provider role is notification only (plus report-link upload and
# account verification). No production/video/publish operations.
KNOWN_OPERATION_CANDIDATES = {
    "account": ["getAccount"],
    "send_notification": ["sendTelegramMessage"],
    "upload_asset": ["uploadAsset"],
    "upload_html_report": ["uploadAsset"],
    "analytics": ["getAnalytics"],
}

CAPABILITY_GROUP_ALIASES = {
    "notification": ["send_notification"],
    "media": ["upload_asset"],
    "analytics": ["account", "analytics"],
}


class ProviderError(RuntimeError):
    pass


def _read_json(path: str | None) -> dict[str, Any]:
    if not path:
        return {}
    p = Path(path)
    if not p.exists():
        return {}
    try:
        data = json.loads(p.read_text(encoding="utf-8"))
    except (ValueError, OSError) as e:
        # a corrupted/hand-edited config is a ProviderError (caught by main -> exit 2),
        # never an uncaught crash; notify's own gates then degrade to local_path_only.
        raise ProviderError(f"provider_config_unreadable: {e}")
    return data if isinstance(data, dict) else {}


def _write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def _provider_block(config: dict[str, Any], provider: str) -> dict[str, Any]:
    providers = config.get("providers", {})
    block = providers.get(provider, {}) if isinstance(providers, dict) else {}
    return block if isinstance(block, dict) else {}  # a non-dict block degrades, never crashes


def _active_provider(config: dict[str, Any], fallback: str = "widecast") -> str:
    return str(config.get("active_provider") or fallback)


def _api_key(config: dict[str, Any], provider: str) -> str:
    block = _provider_block(config, provider)
    generic = os.environ.get("OUTREACHCRM_PROVIDER_API_KEY")
    if generic:
        return generic
    env_name = block.get("api_key_env")
    if env_name and os.environ.get(env_name):
        return os.environ[env_name]
    local = block.get("api_key_local")
    if local:
        return str(local)
    raise ProviderError("provider_auth_missing: no api_key_env value resolved and api_key_local is empty")


def _discovery_url(defaults: dict[str, Any], config: dict[str, Any], provider: str) -> str:
    block = _provider_block(config, provider)
    if block.get("discovery_url"):
        return str(block["discovery_url"])
    default_block = defaults.get("providers", {}).get(provider, {})
    return str(default_block.get("discovery_url") or DEFAULT_DISCOVERY_URL)


def _normalize_url(url: str) -> str:
    return str(url or "").strip().rstrip("/")


def _as_string_list(value: Any) -> list[str]:
    if value is None:
        return []
    if isinstance(value, str):
        return [value]
    if isinstance(value, list):
        return [str(item) for item in value if str(item).strip()]
    return []


def _disabled_server_urls(defaults: dict[str, Any], config: dict[str, Any], provider: str) -> set[str]:
    default_block = defaults.get("providers", {}).get(provider, {})
    block = _provider_block(config, provider)
    disabled = set(DEFAULT_DISABLED_SERVER_URLS.get(provider, set()))
    disabled.update(_as_string_list(default_block.get("disabled_server_urls")))
    disabled.update(_as_string_list(block.get("disabled_server_urls")))
    return {_normalize_url(url) for url in disabled if _normalize_url(url)}


def _preferred_server_urls(defaults: dict[str, Any], config: dict[str, Any], provider: str) -> list[str]:
    default_block = defaults.get("providers", {}).get(provider, {})
    block = _provider_block(config, provider)
    preferred = []
    for source in (
        block.get("server_url"),
        block.get("preferred_server_url"),
        default_block.get("server_url"),
        default_block.get("preferred_server_url"),
    ):
        preferred.extend(_as_string_list(source))
    if provider == "widecast":
        preferred.append(DEFAULT_WIDECAST_SERVER_URL)
    seen = set()
    result = []
    for url in preferred:
        normalized = _normalize_url(url)
        if normalized and normalized not in seen:
            seen.add(normalized)
            result.append(normalized)
    return result


def _select_server_url(
    defaults: dict[str, Any],
    config: dict[str, Any],
    provider: str,
    server_urls: list[str],
) -> tuple[str, list[str]]:
    disabled = _disabled_server_urls(defaults, config, provider)
    cleaned_candidates = []
    seen = set()
    for url in server_urls:
        normalized = _normalize_url(url)
        if normalized and normalized not in seen:
            seen.add(normalized)
            cleaned_candidates.append(normalized)

    skipped_disabled = [url for url in cleaned_candidates if url in disabled]
    for url in _preferred_server_urls(defaults, config, provider):
        if url in disabled:
            if url not in skipped_disabled:
                skipped_disabled.append(url)
            continue
        return url, skipped_disabled

    for url in cleaned_candidates:
        if url in disabled:
            continue
        return url, skipped_disabled

    if skipped_disabled:
        raise ProviderError(
            "provider_discovery_failed: all discovered/preferred OpenAPI server URLs are disabled: "
            + ", ".join(skipped_disabled)
        )
    raise ProviderError("provider_discovery_failed: OpenAPI server URL not found")


def _fetch(url: str, headers: dict[str, str] | None = None) -> tuple[int, str, bytes]:
    merged = {"User-Agent": USER_AGENT, "Accept": "application/yaml, application/json, text/yaml, */*"}
    merged.update(headers or {})
    req = urllib.request.Request(url, headers=merged)
    try:
        with urllib.request.urlopen(req, timeout=45) as resp:
            return resp.status, resp.headers.get("content-type", ""), resp.read()
    except urllib.error.HTTPError as exc:
        body = exc.read()
        raise ProviderError(f"http_error: status={exc.code} body={body[:500].decode('utf-8', 'replace')}")
    except urllib.error.URLError as exc:
        raise ProviderError(f"network_error: {exc.reason}")


def _request_json(
    method: str,
    url: str,
    api_key: str | None,
    body: dict[str, Any] | None = None,
) -> tuple[int, dict[str, str], Any]:
    data = None
    headers = {"Accept": "application/json", "User-Agent": USER_AGENT}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method.upper())
    try:
        with urllib.request.urlopen(req, timeout=90) as resp:
            raw = resp.read()
            text = raw.decode("utf-8", "replace")
            parsed = json.loads(text) if text else {}
            return resp.status, dict(resp.headers), parsed
    except urllib.error.HTTPError as exc:
        text = exc.read().decode("utf-8", "replace")
        try:
            parsed = json.loads(text)
        except json.JSONDecodeError:
            parsed = {"raw": text}
        return exc.code, dict(exc.headers), parsed


def _parse_openapi_yaml(text: str) -> dict[str, Any]:
    server_urls: list[str] = []
    operations: dict[str, dict[str, str]] = {}
    current_path = ""
    current_method = ""
    in_servers = False
    in_paths = False

    for raw in text.splitlines():
        line = raw.rstrip()
        stripped = line.strip()
        if stripped == "servers:":
            in_servers = True
            continue
        if in_servers and stripped.startswith("- url:"):
            server_urls.append(stripped.split(":", 1)[1].strip().strip('"').strip("'"))
            continue
        if stripped == "paths:":
            in_paths = True
            in_servers = False
            continue
        if not in_paths:
            continue

        path_match = re.match(r"^  (/[^\s:]+):\s*$", line)
        if path_match:
            current_path = path_match.group(1)
            current_method = ""
            continue

        method_match = re.match(r"^    (get|post|put|patch|delete):\s*$", line)
        if method_match:
            current_method = method_match.group(1)
            continue

        op_match = re.match(r"^\s+operationId:\s*([A-Za-z0-9_:-]+)\s*$", line)
        if op_match and current_path and current_method:
            operation_id = op_match.group(1).strip().strip('"').strip("'")
            operations[operation_id] = {
                "method": current_method.upper(),
                "path": current_path,
            }

    if not server_urls:
        raise ProviderError("provider_discovery_failed: OpenAPI server URL not found")
    return {"server_urls": server_urls, "server_url": server_urls[0], "operations": operations}


def _parse_openapi(raw: bytes, content_type: str) -> tuple[dict[str, Any], str]:
    text = raw.decode("utf-8", "replace")
    if "json" in content_type or text.lstrip().startswith("{"):
        doc = json.loads(text)
        server_urls = [str(item.get("url")) for item in doc.get("servers", []) if item.get("url")]
        operations = {}
        for path, methods in doc.get("paths", {}).items():
            for method, op in methods.items():
                if str(method).lower() not in HTTP_METHODS:
                    continue
                operation_id = op.get("operationId")
                if operation_id:
                    operations[operation_id] = {"method": method.upper(), "path": path}
        if not server_urls:
            raise ProviderError("provider_discovery_failed: OpenAPI server URL not found")
        return {"server_urls": server_urls, "server_url": server_urls[0], "operations": operations}, text
    return _parse_openapi_yaml(text), text


def _load_spec(args: argparse.Namespace, config: dict[str, Any], defaults: dict[str, Any]) -> tuple[str, dict[str, Any], str]:
    provider = _active_provider(config, args.provider)
    url = args.discovery_url or _discovery_url(defaults, config, provider)
    status, content_type, raw = _fetch(url)
    if status >= 400:
        raise ProviderError(f"provider_discovery_failed: status={status}")
    parsed, raw_text = _parse_openapi(raw, content_type)
    parsed["server_url"], skipped_disabled = _select_server_url(
        defaults,
        config,
        provider,
        parsed.get("server_urls") or [parsed.get("server_url", "")],
    )
    parsed["skipped_disabled_server_urls"] = skipped_disabled
    return provider, parsed, raw_text


def _operation_aliases(operations: dict[str, dict[str, str]]) -> dict[str, str]:
    by_lower = {operation_id.lower(): operation_id for operation_id in operations}
    aliases: dict[str, str] = {}
    for alias, candidates in KNOWN_OPERATION_CANDIDATES.items():
        for candidate in candidates:
            operation_id = by_lower.get(candidate.lower())
            if operation_id:
                aliases[alias] = operation_id
                break
    return aliases


def _capability_status(operation_aliases: dict[str, str]) -> dict[str, str]:
    status: dict[str, str] = {}
    for group, required_aliases in CAPABILITY_GROUP_ALIASES.items():
        present = [alias for alias in required_aliases if alias in operation_aliases]
        if len(present) == len(required_aliases):
            status[group] = "available"
        elif present:
            status[group] = "partial"
        else:
            status[group] = "unavailable"
    return status


def _missing_capability_aliases(operation_aliases: dict[str, str]) -> dict[str, list[str]]:
    missing: dict[str, list[str]] = {}
    for group, required_aliases in CAPABILITY_GROUP_ALIASES.items():
        group_missing = [alias for alias in required_aliases if alias not in operation_aliases]
        if group_missing:
            missing[group] = group_missing
    return missing


def _url_for(server_url: str, path: str, query: list[str] | None = None) -> str:
    url = server_url.rstrip("/") + "/" + path.lstrip("/")
    if query:
        pairs = []
        for item in query:
            if "=" not in item:
                raise ProviderError(f"bad_query: expected key=value, got {item}")
            k, v = item.split("=", 1)
            pairs.append((k, v))
        url += "?" + urllib.parse.urlencode(pairs)
    return url


def _write_call_log(config_path: str | None, provider: str, operation: str, status: int, blocker: str = "") -> None:
    if not config_path:
        return
    config_file = Path(config_path)
    log_path = config_file.with_name("provider_calls.jsonl")
    record = {
        "ts": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "provider": provider,
        "operationId": operation,
        "status": status,
        "blocker": blocker,
    }
    log_path.parent.mkdir(parents=True, exist_ok=True)
    with log_path.open("a", encoding="utf-8") as fh:
        fh.write(json.dumps(record, ensure_ascii=False) + "\n")


def cmd_discover(args: argparse.Namespace) -> int:
    config = _read_json(args.config)
    defaults = _read_json(args.defaults)
    provider, parsed, raw_text = _load_spec(args, config, defaults)
    operation_aliases = _operation_aliases(parsed["operations"])
    out = {
        "schema_version": 1,
        "provider": provider,
        "discovered_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "discovery_url": args.discovery_url or _discovery_url(defaults, config, provider),
        "server_url": parsed["server_url"],
        "server_urls_discovered": parsed.get("server_urls", []),
        "server_urls_skipped_disabled": parsed.get("skipped_disabled_server_urls", []),
        "operation_ids": {k: v for k, v in sorted(parsed["operations"].items())},
        "operation_aliases": {k: operation_aliases[k] for k in sorted(operation_aliases)},
        "capability_status": _capability_status(operation_aliases),
        "missing_capability_aliases": _missing_capability_aliases(operation_aliases),
    }
    if args.out_dir:
        out_dir = Path(args.out_dir)
        _write_json(out_dir / "provider_capabilities.json", out)
        (out_dir / "provider_openapi_cache.yaml").write_text(raw_text, encoding="utf-8")
    print(json.dumps(out, indent=2, ensure_ascii=False))
    return 0


def cmd_call(args: argparse.Namespace) -> int:
    config = _read_json(args.config)
    defaults = _read_json(args.defaults)
    provider, parsed, _ = _load_spec(args, config, defaults)
    operation = parsed["operations"].get(args.operation)
    if not operation:
        raise ProviderError(f"provider_required_operation_missing: {args.operation}")
    body = json.loads(args.body) if args.body else None
    url = _url_for(parsed["server_url"], operation["path"], args.query)
    key = None if args.no_auth else _api_key(config, provider)
    status, headers, response = _request_json(operation["method"], url, key, body)
    blocker = "" if status < 400 else f"provider_call_failed: status={status}"
    _write_call_log(args.config, provider, args.operation, status, blocker)
    print(json.dumps({"status": status, "headers": _selected_headers(headers), "body": response}, indent=2, ensure_ascii=False))
    return 0 if status < 400 else 1


def _selected_headers(headers: dict[str, str]) -> dict[str, str]:
    keep = {}
    for key, value in headers.items():
        lk = key.lower()
        if lk in {"x-request-id", "x-ratelimit-limit", "x-ratelimit-remaining", "x-ratelimit-reset", "x-widecast-version"}:
            keep[key] = value
    return keep


def cmd_account(args: argparse.Namespace) -> int:
    args.operation = args.operation or "getAccount"
    args.body = None
    args.query = []
    return cmd_call(args)


def cmd_upload_report(args: argparse.Namespace) -> int:
    report_path = Path(args.file)
    if not report_path.exists():
        raise ProviderError(f"file_not_found: {report_path}")
    encoded = base64.b64encode(report_path.read_bytes()).decode("ascii")
    body = {
        "file_data": encoded,
        "filename": report_path.name,
        "content_type": args.content_type,
    }
    args.operation = args.operation or "uploadAsset"
    args.body = json.dumps(body)
    args.query = []
    return cmd_call(args)


# --- composed operator notification (discover -> verify -> upload -> send) ----

_NOTIFY_LOG_COLUMNS = [
    "Date", "Agent", "Event", "Channel", "Status", "Report Path", "Report Link Sent",
    "Provider", "Provider Discovery Checked", "Upload Operation", "Notification Operation",
    "Upload Attempted", "Uploaded Report URL", "Notification Attempted", "Blocker", "Action Needed",
]


def _append_notification_log(log_path: str, row: dict[str, str]) -> None:
    """Append one row to notifications/notification_log.md (creating the header if absent),
    matching the 16-column format documented in playbooks/07 §11.8."""
    p = Path(log_path)
    p.parent.mkdir(parents=True, exist_ok=True)
    if not p.exists() or not p.read_text(encoding="utf-8").strip():
        header = "# Notification Log\n\n| " + " | ".join(_NOTIFY_LOG_COLUMNS) + " |\n" \
                 + "|" + "|".join(["---"] * len(_NOTIFY_LOG_COLUMNS)) + "|\n"
        p.write_text(header, encoding="utf-8")
    cells = [str(row.get(c, "") or "").replace("|", "\\|").replace("\n", " ") for c in _NOTIFY_LOG_COLUMNS]
    with p.open("a", encoding="utf-8") as fh:
        fh.write("| " + " | ".join(cells) + " |\n")


def _notification_block(config: dict[str, Any], provider: str) -> dict[str, Any]:
    block = _provider_block(config, provider).get("notification", {})
    return block if isinstance(block, dict) else {}


def _notification_enabled(config: dict[str, Any], provider: str) -> bool:
    return bool(_notification_block(config, provider).get("enabled"))


def cmd_notify(args: argparse.Namespace) -> int:
    """Compose the operator notification in one deterministic step:
    config check -> (dry-run/degraded short-circuit) -> discover -> getAccount ->
    optional uploadAsset(report) -> sendTelegramMessage -> notification_log.md row.
    A missing/disabled provider is a valid degraded outcome (`local_path_only`, exit 0),
    never a run failure. `--dry-run` does everything except touch the network or send."""
    config = _read_json(args.config)
    defaults = _read_json(args.defaults)
    provider = _active_provider(config, args.provider)
    now = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    row = {"Date": now[:10], "Agent": args.agent, "Event": args.event,
           "Channel": "WideCast Telegram/email fallback", "Provider": provider,
           "Report Path": args.report_path or (args.report_file or ""),
           "Action Needed": args.action_needed,
           "Provider Discovery Checked": "no", "Upload Attempted": "no",
           "Notification Attempted": "no", "Report Link Sent": "no"}

    def finish(status: str, blocker: str = "", exit_code: int = 0, **extra) -> int:
        row["Status"] = status
        row["Blocker"] = blocker or "none"
        row.update(extra)
        if args.log:
            _append_notification_log(args.log, row)
        out = {"status": status, "blocker": blocker or None, "provider": provider,
               "event": args.event, "dry_run": bool(args.dry_run)}
        out.update({k.lower().replace(" ", "_"): v for k, v in extra.items()})
        print(json.dumps(out, indent=2, ensure_ascii=False))
        return exit_code

    # 1. Configuration gates — a degraded state is honest and non-fatal.
    if not config or not _provider_block(config, provider):
        return finish("local_path_only", "provider_config_missing")
    if not _notification_enabled(config, provider):
        return finish("local_path_only", "provider_notification_not_configured")
    try:
        _api_key(config, provider)
    except ProviderError:
        return finish("local_path_only", "provider_auth_missing")

    # 2. Dry-run: report the plan, no network, no send.
    if args.dry_run:
        return finish("dry_run", "",
                      **{"Notification Operation": "sendTelegramMessage",
                         "Upload Operation": "uploadAsset" if args.report_file else ""})

    # 3. Real path: discover -> verify -> upload -> send.
    try:
        _, parsed, _ = _load_spec(args, config, defaults)
    except ProviderError as exc:
        return finish("blocked", str(exc).split(":")[0] or "provider_discovery_failed", 1)
    row["Provider Discovery Checked"] = "yes"
    aliases = _operation_aliases(parsed["operations"])
    if "send_notification" not in aliases:
        return finish("blocked", "provider_required_operation_missing", 1)
    key = _api_key(config, provider)
    row["Notification Operation"] = aliases["send_notification"]

    notif_cfg = _notification_block(config, provider)
    upload_cfg = _provider_block(config, provider).get("report_upload", {})
    upload_cfg = upload_cfg if isinstance(upload_cfg, dict) else {}
    # provider response-URL keys are schema-specific; overridable so a live schema matches w/o a code change
    url_fields = upload_cfg.get("url_fields") or ["url", "asset_url", "link", "download_url"]
    uploaded_url = ""
    if args.report_file:
        row["Upload Operation"] = aliases.get("upload_asset", "uploadAsset")
        if "upload_asset" not in aliases:
            # the OPTIONAL upload op is absent; distinct from the hard send-op-missing failure —
            # send text-only, note softly (a monitor for real failures must not fire here)
            row["Blocker"] = "provider_upload_operation_missing"
        elif not Path(args.report_file).exists():
            row["Blocker"] = "provider_upload_failed"
        else:
            row["Upload Attempted"] = "yes"
            up = parsed["operations"][aliases["upload_asset"]]
            body = {"file_data": base64.b64encode(Path(args.report_file).read_bytes()).decode("ascii"),
                    "filename": Path(args.report_file).name, "content_type": "text/html"}
            st, _, resp = _request_json(up["method"], _url_for(parsed["server_url"], up["path"]), key, body)
            if st >= 400:
                row["Blocker"] = "provider_upload_failed"  # the HTTP upload itself failed
            elif isinstance(resp, dict):
                uploaded_url = next((str(resp[k]) for k in url_fields if resp.get(k)), "")
                if not uploaded_url:
                    # upload SUCCEEDED but the URL key wasn't recognized — distinct from a real failure
                    row["Blocker"] = "provider_upload_url_unrecognized"

    op = parsed["operations"][aliases["send_notification"]]
    text = args.message + (f"\n\nReport: {uploaded_url}" if uploaded_url else "")
    # The message body property name is provider-schema-specific; default "text", overridable in
    # provider_config notification.text_field so a live schema can be matched without a code change.
    text_field = notif_cfg.get("text_field") or "text"
    row["Notification Attempted"] = "yes"
    st, _, resp = _request_json(op["method"], _url_for(parsed["server_url"], op["path"]), key, {text_field: text})
    _write_call_log(args.config, provider, aliases["send_notification"], st,
                    "" if st < 400 else "provider_notification_failed")
    if st >= 400:
        return finish("blocked", "provider_notification_failed", 1,
                      **{"Uploaded Report URL": uploaded_url, "Report Link Sent": "yes" if uploaded_url else "no"})
    return finish("sent", row.get("Blocker", "") if row.get("Blocker") not in (None, "", "none") else "",
                  **{"Uploaded Report URL": uploaded_url, "Report Link Sent": "yes" if uploaded_url else "no"})


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="OutreachCRM OpenAPI provider adapter")
    parser.add_argument("--config", help="Path to per-client provider_config.local.json")
    parser.add_argument("--defaults", help="Path to daily-content-pipeline/provider_defaults.json")
    parser.add_argument("--provider", default="widecast")
    parser.add_argument("--discovery-url")
    sub = parser.add_subparsers(dest="cmd", required=True)

    discover = sub.add_parser("discover", help="Fetch OpenAPI spec and list operations")
    discover.add_argument("--out-dir", help="Write provider_capabilities.json and provider_openapi_cache.yaml")
    discover.set_defaults(func=cmd_discover)

    call = sub.add_parser("call", help="Call an operationId with optional JSON body")
    call.add_argument("--operation", required=True)
    call.add_argument("--body", help="JSON request body")
    call.add_argument("--query", action="append", help="Query param as key=value")
    call.add_argument("--no-auth", action="store_true", help="Do not send Authorization header")
    call.set_defaults(func=cmd_call)

    account = sub.add_parser("account", help="Verify account using getAccount")
    account.add_argument("--operation", default="getAccount")
    account.add_argument("--no-auth", action="store_true")
    account.set_defaults(func=cmd_account)

    upload = sub.add_parser("upload-report", help="Upload an HTML report through uploadAsset")
    upload.add_argument("--file", required=True)
    upload.add_argument("--operation", default="uploadAsset")
    upload.add_argument("--content-type", default="text/html")
    upload.add_argument("--no-auth", action="store_true")
    upload.set_defaults(func=cmd_upload_report)

    notify = sub.add_parser("notify", help="Composed operator notification: verify -> optional upload -> send -> log")
    notify.add_argument("--message", required=True, help="Operator-facing status text (counts + report link).")
    notify.add_argument("--event", default="daily_run_completed",
                        choices=["daily_run_completed", "weekly_client_report_ready"])
    notify.add_argument("--report-file", help="HTML report to uploadAsset for the report link.")
    notify.add_argument("--report-path", help="Human-readable report path to log (defaults to --report-file).")
    notify.add_argument("--log", help="Path to notifications/notification_log.md to append a row.")
    notify.add_argument("--agent", default="Claude Schedule", help="Agent column for the log row.")
    notify.add_argument("--action-needed", default="", help="Action Needed column for the log row.")
    notify.add_argument("--dry-run", action="store_true", help="Resolve + report the plan; no network, no send.")
    notify.set_defaults(func=cmd_notify)
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    try:
        return args.func(args)
    except ProviderError as exc:
        print(str(exc), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
