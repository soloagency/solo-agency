#!/usr/bin/env python3
"""Minimal OpenAPI provider adapter for Solo Agency.

This utility intentionally uses only Python's standard library so scheduled
agents can run it without installing dependencies. It supports the subset Solo
Agency needs first: discover operation IDs, verify account, call JSON endpoints,
and upload HTML reports through a provider operation such as WideCast uploadAsset.
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
USER_AGENT = "SoloAgencyOpenAPIAdapter/1.0"

KNOWN_OPERATION_CANDIDATES = {
    "account": ["getAccount"],
    "analytics": ["getAnalytics"],
    "list_videos": ["listVideos"],
    "upload_asset": ["uploadAsset"],
    "upload_html_report": ["uploadAsset"],
    "send_notification": ["sendNotification", "sendTelegramMessage"],
    "publish": ["publish"],
    "create_video": ["createVideo"],
    "export_video": ["exportVideo"],
    "get_status": ["getStatus", "waitForVideo"],
    "get_video_data": ["getVideoData", "videoData"],
    "get_writing_skill": ["getWritingSkill"],
    "get_editing_skill": ["getEditingSkill"],
    "create_content": ["createContent"],
    "create_image": ["createImage"],
    "search_broll": ["searchBroll"],
    "collect_ideas": ["collectIdeas"],
    "scene_geometry": ["sceneGeometry", "getSceneGeometry"],
    "scene_inspector": ["sceneInspector", "inspectScene", "getSceneInspector"],
    "modify_scene": ["modifyScene"],
}

CAPABILITY_GROUP_ALIASES = {
    "production": ["create_video", "get_status"],
    "video_editing": [
        "get_editing_skill",
        "get_video_data",
        "scene_geometry",
        "scene_inspector",
        "modify_scene",
    ],
    "render_export": ["export_video"],
    "media": ["upload_asset", "create_image", "search_broll"],
    "distribution": ["publish"],
    "notification": ["send_notification"],
    "analytics": ["account", "analytics", "list_videos", "get_status", "get_video_data"],
}


class ProviderError(RuntimeError):
    pass


def _read_json(path: str | None) -> dict[str, Any]:
    if not path:
        return {}
    p = Path(path)
    if not p.exists():
        return {}
    return json.loads(p.read_text(encoding="utf-8"))


def _write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def _provider_block(config: dict[str, Any], provider: str) -> dict[str, Any]:
    return config.get("providers", {}).get(provider, {})


def _active_provider(config: dict[str, Any], fallback: str = "widecast") -> str:
    return str(config.get("active_provider") or fallback)


def _api_key(config: dict[str, Any], provider: str) -> str:
    block = _provider_block(config, provider)
    generic = os.environ.get("SOLO_AGENCY_PROVIDER_API_KEY")
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


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Solo Agency OpenAPI provider adapter")
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
