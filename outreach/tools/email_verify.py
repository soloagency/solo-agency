#!/usr/bin/env python3
"""email_verify.py — cheap, local email verification: syntax + MX only.

Deliberately does NOT do an SMTP RCPT probe (residential ISPs block port 25, big
providers are catch-all, and probing looks like a spammer — DESIGN §9.6). MX proves
the domain can receive mail, not that the mailbox exists; guessed/pattern addresses
still need a third-party verification API (a later phase). Stdlib only: MX lookup via
`dig`, falling back to `nslookup`, then to an A-record check.

  email_verify.py check --email a@b.com
  email_verify.py check-file --in emails.txt        # one address per line -> JSONL to stdout
"""

from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
import sys
import socket

_SYNTAX = re.compile(r"^[A-Za-z0-9!#$%&'*+/=?^_`{|}~.-]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+$")


def syntax_ok(email: str) -> bool:
    email = (email or "").strip()
    if not email or email.count("@") != 1 or ".." in email:
        return False
    if len(email) > 254:
        return False
    return bool(_SYNTAX.match(email))


def mx_lookup(domain: str, timeout: float = 5.0) -> list[str]:
    """Return MX hostnames (may be empty). Tries dig, then nslookup, then A-record."""
    dig = shutil.which("dig")
    if dig:
        try:
            out = subprocess.run([dig, "+short", "MX", domain], capture_output=True,
                                 text=True, timeout=timeout)
            hosts = []
            for line in out.stdout.splitlines():
                parts = line.split()
                if len(parts) == 2:  # "10 mx.example.com."
                    hosts.append(parts[1].rstrip("."))
            if hosts:
                return hosts
        except (subprocess.SubprocessError, OSError):
            pass
    ns = shutil.which("nslookup")
    if ns:
        try:
            out = subprocess.run([ns, "-query=MX", domain], capture_output=True,
                                 text=True, timeout=timeout)
            hosts = []
            for line in out.stdout.splitlines():
                m = re.search(r"mail exchanger = (\S+)", line)
                if m:
                    hosts.append(m.group(1).rstrip("."))
            if hosts:
                return hosts
        except (subprocess.SubprocessError, OSError):
            pass
    # last resort: does the domain resolve at all (A/AAAA)? implicit MX per RFC 5321.
    try:
        socket.getaddrinfo(domain, None)
        return [domain]  # implicit MX
    except (socket.gaierror, OSError):
        return []


def check(email: str) -> dict:
    email = (email or "").strip()
    syn = syntax_ok(email)
    result = {"email": email, "syntax_ok": syn, "mx_ok": False, "mx_hosts": [],
              "status": "unverified", "note": ""}
    if not syn:
        result["status"] = "syntax_invalid"
        result["note"] = "address failed syntax check"
        return result
    domain = email.rsplit("@", 1)[1].lower()
    hosts = mx_lookup(domain)
    result["mx_hosts"] = hosts
    result["mx_ok"] = bool(hosts)
    if hosts:
        result["status"] = "mx_ok"
        result["note"] = "domain can receive mail (mailbox existence NOT proven; guessed addresses still need API verification)"
    else:
        result["status"] = "mx_fail"
        result["note"] = "no MX and domain does not resolve — will hard-bounce"
    return result


def main(argv=None) -> int:
    p = argparse.ArgumentParser(description="Cheap local email verification (syntax + MX)")
    sub = p.add_subparsers(dest="cmd", required=True)
    c = sub.add_parser("check"); c.add_argument("--email", required=True)
    cf = sub.add_parser("check-file"); cf.add_argument("--in", dest="infile", required=True)
    args = p.parse_args(argv)
    if args.cmd == "check":
        print(json.dumps(check(args.email), ensure_ascii=False))
        return 0
    if args.cmd == "check-file":
        with open(args.infile, "r", encoding="utf-8") as fh:
            for line in fh:
                addr = line.strip()
                if addr:
                    print(json.dumps(check(addr), ensure_ascii=False))
        return 0
    return 2


if __name__ == "__main__":
    sys.exit(main())
