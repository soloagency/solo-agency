#!/usr/bin/env python3
"""Render plan numbers from plans.json into the playbooks — the single-source-of-truth mechanism.

plans.json (repo root) is the ONLY file where a plan number lives. This script:

  --write [--only FILE ...]  rewrites playbooks/PLANS.md and the body of every
                             <!--plan:KEY-->…<!--/plan--> token in every tracked .md file
                             (or only the files given), from plans.json.
  --check                    exits 1 when PLANS.md or any token is stale, when plans.json is invalid,
                             or when a plan number is typed anywhere outside a token (the guard).
  --show                     prints every KEY and its rendered value.
  --validate                 only validates plans.json (used by the deploy scripts before copying it).

Token keys: <tier>.name | <tier>.price | <tier>.price_usd | <tier>.max_contacts | <tier>.seats |
<tier>.write_actions | <tier>.<any limit> | ladder_line | caps_line | caps_slash | caps_named_line |
names_line | prices_line | paid_prices_line | paid_caps_line | next_after_free.(name|price|max_contacts) |
first_write_actions_tier.name | approaching_ratio | approaching_percent | upgrade_url | version | updated | table.

A line that legitimately carries a number next to a tier word may end with `<!--plan:ok-->` to be
skipped by the guard. deploy-soloagency.sh runs --write before the LOAD_MANIFEST step and --check in
its static checks, so a typed number anywhere else refuses to publish.
"""
import argparse
import json
import os
import re
import subprocess
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PLANS_PATH = os.path.join(ROOT, "plans.json")
PLANS_MD = os.path.join("playbooks", "PLANS.md")
LIMIT_KEYS = ["max_contacts", "max_clients", "max_groups", "max_campaigns", "max_sends_per_day"]
TIER_ID_RE = re.compile(r"^[a-z][a-z0-9_-]{0,31}$")
TOKEN_RE = re.compile(r"<!--plan:([A-Za-z0-9_.]+)-->(.*?)<!--/plan-->", re.S)
OK_MARK = "<!--plan:ok-->"
# Files the guard never scans: the rendered copy itself, local handoffs, backups, test fixtures.
SKIP_RE = re.compile(
    r"(^|/)(PLANS\.md|HANDOFF_LOCAL[^/]*\.md|LOAD_MANIFEST\.md)$"
    r"|_20[0-9]{2}-[0-9]{2}-[0-9]{2}_[0-9]{2}-[0-9]{2}-[0-9]{2}"
    r"|_20[0-9]{6}_[0-9]{4}"
    r"|(^|/)(node_modules|dist|\.git|tests|testdata|_backup|backup|backups)/"
)


def fail(msg):
    sys.stderr.write("render_plans: " + msg + "\n")
    sys.exit(1)


def load_plans(path=PLANS_PATH):
    try:
        with open(path, encoding="utf-8") as fh:
            doc = json.load(fh)
    except Exception as e:  # noqa: BLE001
        fail("%s: cannot read/parse: %s" % (path, e))
    tiers = doc.get("tiers")
    if not isinstance(tiers, list) or not tiers:
        fail("tiers must be a non-empty list")
    ids, ranks = set(), set()
    for t in tiers:
        for k in ("id", "name", "rank", "price_usd", "widecast_paid", "seats", "features", "limits"):
            if k not in t:
                fail("tier %r is missing %r" % (t.get("id"), k))
        tid = t["id"]
        if not isinstance(tid, str) or not TIER_ID_RE.match(tid):
            fail("tier id %r must match %s" % (tid, TIER_ID_RE.pattern))
        if tid in ids:
            fail("duplicate tier id %r" % tid)
        ids.add(tid)
        if not isinstance(t["rank"], int) or t["rank"] < 0 or t["rank"] in ranks:
            fail("tier %r: rank must be a unique non-negative integer" % tid)
        ranks.add(t["rank"])
        if not isinstance(t["price_usd"], int) or t["price_usd"] < 0:
            fail("tier %r: price_usd must be a non-negative integer" % tid)
        if not isinstance(t["seats"], int) or t["seats"] < 0:
            fail("tier %r: seats must be a non-negative integer" % tid)
        if not isinstance(t["widecast_paid"], list) or not all(isinstance(p, int) for p in t["widecast_paid"]):
            fail("tier %r: widecast_paid must be a list of integers" % tid)
        if not isinstance(t["features"], list) or not all(isinstance(f, str) and f for f in t["features"]):
            fail("tier %r: features must be a list of non-empty strings" % tid)
        limits = t["limits"]
        if not isinstance(limits, dict):
            fail("tier %r: limits must be an object" % tid)
        for k in LIMIT_KEYS:
            if not isinstance(limits.get(k), int) or limits[k] < 0:
                fail("tier %r: limits.%s must be a non-negative integer (0 = unlimited)" % (tid, k))
    if "free" not in ids:
        fail("a tier with id 'free' is required (the keyless / offline default)")
    ratio = doc.get("approaching_ratio")
    if not isinstance(ratio, (int, float)) or not (0 < float(ratio) <= 1):
        fail("approaching_ratio must be a number in (0, 1]")
    if not isinstance(doc.get("upgrade_url"), str) or not doc["upgrade_url"]:
        fail("upgrade_url must be a non-empty string")
    doc["tiers"] = sorted(tiers, key=lambda t: t["rank"])
    return doc


def fmt_cap(v):
    return "unlimited" if v == 0 else str(v)


def fmt_cap_short(v):
    return "∞" if v == 0 else str(v)


def fmt_price(t):
    if t.get("contact_sales"):
        return "contact us"
    return "$%d" % t["price_usd"]


def has_write_actions(t):
    return "write_actions" in t["features"]


def render_keys(doc):
    tiers = doc["tiers"]
    keys = {
        "version": str(doc.get("version", "")),
        "updated": str(doc.get("updated", "")),
        "upgrade_url": doc["upgrade_url"],
        "approaching_ratio": ("%g" % float(doc["approaching_ratio"])),
        "approaching_percent": ("%d%%" % round(float(doc["approaching_ratio"]) * 100)),
    }
    for t in tiers:
        p = t["id"] + "."
        keys[p + "name"] = t["name"]
        keys[p + "price"] = fmt_price(t)
        keys[p + "price_usd"] = str(t["price_usd"])
        keys[p + "rank"] = str(t["rank"])
        keys[p + "seats"] = str(t["seats"])
        keys[p + "write_actions"] = "yes" if has_write_actions(t) else "no"
        for k in LIMIT_KEYS:
            keys[p + k] = fmt_cap(t["limits"][k])
    free = next(t for t in tiers if t["id"] == "free")
    paid = [t for t in tiers if t["rank"] > free["rank"]]

    def ladder_item(t):
        if t.get("contact_sales"):
            return "%s (contact us) → %s" % (t["name"], fmt_cap(t["limits"]["max_contacts"]))
        if t["price_usd"] == 0:
            return "%s %s" % (t["name"], fmt_cap(t["limits"]["max_contacts"]))
        return "%s $%d → %s" % (t["name"], t["price_usd"], fmt_cap(t["limits"]["max_contacts"]))

    def priced_name(t):
        if t["price_usd"] == 0 or t.get("contact_sales"):
            return t["name"]
        return "%s $%d" % (t["name"], t["price_usd"])

    keys["ladder_line"] = " · ".join(ladder_item(t) for t in tiers)
    keys["caps_line"] = " / ".join(fmt_cap(t["limits"]["max_contacts"]) for t in tiers)
    keys["caps_slash"] = "/".join(fmt_cap_short(t["limits"]["max_contacts"]) for t in tiers)
    keys["caps_named_line"] = ", ".join("%s %s" % (t["name"], fmt_cap(t["limits"]["max_contacts"])) for t in tiers)
    keys["names_line"] = " · ".join(t["name"] for t in tiers)
    keys["prices_line"] = " · ".join(priced_name(t) for t in tiers)
    keys["paid_prices_line"] = " / ".join(priced_name(t) for t in paid)
    keys["paid_caps_line"] = ", ".join("%s %s" % (t["name"], fmt_cap(t["limits"]["max_contacts"])) for t in paid)
    if paid:
        nxt = paid[0]
        keys["next_after_free.name"] = nxt["name"]
        keys["next_after_free.price"] = fmt_price(nxt)
        keys["next_after_free.max_contacts"] = fmt_cap(nxt["limits"]["max_contacts"])
    wa = [t for t in tiers if has_write_actions(t)]
    keys["first_write_actions_tier.name"] = wa[0]["name"] if wa else "none"
    keys["table"] = "\n\n" + render_table(doc) + "\n\n"
    return keys


def render_table(doc):
    tiers = doc["tiers"]

    def head(t):
        if t.get("contact_sales"):
            return "%s (contact us)" % t["name"]
        if t["price_usd"] == 0:
            return t["name"]
        return "%s $%d" % (t["name"], t["price_usd"])

    rows = [
        "| | " + " | ".join(head(t) for t in tiers) + " |",
        "|---|" + "---|" * len(tiers),
        "| CRM contacts (stored and unlocked) | " + " | ".join(fmt_cap(t["limits"]["max_contacts"]) for t in tiers) + " |",
        "| Every data feature: enrich, dossier, contact ladder, harvest incl. people search, Zillow, auto update, priority fixes | "
        + " | ".join("yes" if "enrich" in t["features"] else "no" for t in tiers) + " |",
        "| DM to UNLOCKED contacts (`fb.message.send`) | " + " | ".join("yes" for _ in tiers) + " |",
        "| Group post, comment, react (`write_actions`) | " + " | ".join("yes" if has_write_actions(t) else "no" for t in tiers) + " |",
        "| Installs per key (seats) | " + " | ".join(fmt_cap(t["seats"]) for t in tiers) + " |",
    ]
    return "\n".join(rows)


def render_plans_md(doc, keys):
    lines = [
        "# PLANS — the Solo Agency plan ladder (rendered copy)",
        "",
        "Auto-generated from `plans.json` by `tools/render_plans.py` (run by `deploy-soloagency.sh`). Do not edit by hand —",
        "edit `plans.json` and redeploy. Version %s, updated %s." % (keys["version"], keys["updated"]),
        "",
        "**Rule for every brain:** never quote a plan number from memory or from any playbook, this file included. When a",
        "number is about to be spoken to the human, run `<bridge> tool plans show --pipeline daily-content-pipeline` (or read",
        "`GET /status` → `entitlement.ladder`) and quote what it returns; the counts come from",
        "`tool crm-store … contact lock-status`, whose `next_tier` names the plan that raises the cap. The server signs the",
        "same ladder into every entitlement token, so the bridge always speaks the server's numbers.",
        "",
        render_table(doc),
        "",
        "- `0` / unlimited = no cap. The CRM contact cap is the only sold limit: contacts past it are still captured, just",
        "  locked (no detail view, no email, no DM, no campaign) until the plan grows. `max_clients`, `max_groups`,",
        "  `max_campaigns` and `max_sends_per_day` are flat technical ceilings, not plan differentiators.",
        "- `write_actions` (group post, comment, react) is the only plan-gated feature; it starts at %s." % keys["first_write_actions_tier.name"],
        "- Approaching-cap ratio: `%s` (`contact lock-status` → `approaching: true` once `unlocked / max_contacts` reaches it)." % keys["approaching_ratio"],
        "- Upgrade path: %s (the Solo Agency tier follows the WideCast plan on that account)." % keys["upgrade_url"],
        "",
        "Ladder in one line: %s." % keys["ladder_line"],
        "",
    ]
    return "\n".join(lines)


def git_md_files():
    out = subprocess.run(
        ["git", "-C", ROOT, "ls-files", "-co", "--exclude-standard", "--", "*.md"],
        capture_output=True, text=True, check=True,
    ).stdout
    files = sorted(set(l.strip() for l in out.splitlines() if l.strip()))
    # bridge-go is closed source (gitignored *.go) but its README carries tokens too.
    extra = os.path.join("solo-agency-collector", "bridge-go", "README.md")
    if os.path.exists(os.path.join(ROOT, extra)) and extra not in files:
        files.append(extra)
    return [f for f in files if not SKIP_RE.search(f)]


def render_tokens(text, keys, rel):
    problems = []

    def sub(m):
        key = m.group(1)
        if key not in keys:
            problems.append("%s: unknown token key %r" % (rel, key))
            return m.group(0)
        return "<!--plan:%s-->%s<!--/plan-->" % (key, keys[key])

    return TOKEN_RE.sub(sub, text), problems


def guard_patterns(doc):
    names = sorted({t["name"] for t in doc["tiers"]} | {t["name"].capitalize() for t in doc["tiers"]}, key=len, reverse=True)
    alt = "|".join(re.escape(n) for n in names)
    return [
        # a tier name followed within a few characters by a 2-6 digit number: "Free 30", "Starter ($49)", "Pro 2000"
        re.compile(r"\b(?:%s)\b[^\n|\d]{0,16}?\$?\b\d{2,6}\b(?!-\d|\.\d|%%|:\d|/\d)" % alt),
        # a number glued to "contact(s)": "30 CRM contacts", "up to 500 contacts", "30 contact"
        re.compile(r"(?<![{\w])\d{2,6}\s?(?:CRM\s)?contacts?\b"),
        # a sample max_contacts value: "max_contacts: 30"
        re.compile(r"max_contacts\W{0,4}\d{2,6}\b"),
        # a slash ladder: "30/500/2000/10000", "30 / 500 / 2000"
        re.compile(r"(?<![\d/])\d{2,6}\s?/\s?\d{3,6}\s?/\s?\d{3,6}\b"),
    ]


def guard(doc, files):
    pats = guard_patterns(doc)
    hits = []
    for rel in files:
        path = os.path.join(ROOT, rel)
        try:
            text = open(path, encoding="utf-8").read()
        except Exception:  # noqa: BLE001
            continue
        stripped = TOKEN_RE.sub(lambda m: "<!--plan:%s--><!--/plan-->" % m.group(1), text)
        for n, line in enumerate(stripped.splitlines(), 1):
            if OK_MARK in line:
                continue
            for p in pats:
                m = p.search(line)
                if m:
                    hits.append("%s:%d: %r" % (rel, n, m.group(0)))
                    break
    return hits


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--write", action="store_true")
    ap.add_argument("--check", action="store_true")
    ap.add_argument("--show", action="store_true")
    ap.add_argument("--validate", action="store_true")
    ap.add_argument("--only", nargs="*", default=None, help="with --write: only these files (relative to the repo root)")
    args = ap.parse_args()
    doc = load_plans()
    if args.validate and not (args.write or args.check or args.show):
        print("plans.json OK: %s" % ", ".join("%s=%s" % (t["id"], fmt_cap(t["limits"]["max_contacts"])) for t in doc["tiers"]))
        return 0
    keys = render_keys(doc)
    if args.show:
        for k in sorted(keys):
            print("%s = %s" % (k, keys[k].strip()))
        return 0
    files = git_md_files()
    if args.only is not None:
        wanted = {os.path.relpath(os.path.abspath(f), ROOT) for f in args.only}
        files = [f for f in files if f in wanted]
        missing = wanted - set(files)
        if missing:
            fail("--only: not a scannable tracked .md file: %s" % ", ".join(sorted(missing)))
    problems, stale, changed = [], [], []
    plans_md_text = render_plans_md(doc, keys)
    plans_md_path = os.path.join(ROOT, PLANS_MD)
    current_md = open(plans_md_path, encoding="utf-8").read() if os.path.exists(plans_md_path) else None
    if args.only is None and current_md != plans_md_text:
        if args.write:
            os.makedirs(os.path.dirname(plans_md_path), exist_ok=True)
            with open(plans_md_path, "w", encoding="utf-8") as fh:
                fh.write(plans_md_text)
            changed.append(PLANS_MD)
        else:
            stale.append(PLANS_MD)
    for rel in files:
        path = os.path.join(ROOT, rel)
        text = open(path, encoding="utf-8").read()
        if "<!--plan:" not in text:
            continue
        new, probs = render_tokens(text, keys, rel)
        problems.extend(probs)
        if new != text:
            if args.write:
                with open(path, "w", encoding="utf-8") as fh:
                    fh.write(new)
                changed.append(rel)
            else:
                stale.append(rel)
    if args.write:
        for c in changed:
            print("rendered: " + c)
        if not changed:
            print("nothing to render (all tokens current)")
    if args.check:
        hits = guard(doc, files)
        for s in stale:
            problems.append("stale (run --write): " + s)
        for h in hits:
            problems.append("plan number typed outside a <!--plan:…--> token: " + h)
    if problems:
        for p in problems:
            sys.stderr.write("render_plans: " + p + "\n")
        return 1
    if args.check:
        print("render_plans: OK — %d files scanned, no stale token, no typed plan number" % len(files))
    return 0


if __name__ == "__main__":
    sys.exit(main())
