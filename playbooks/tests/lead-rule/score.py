#!/usr/bin/env python3
"""Score a run of lead-qualification-rule judgments against dataset.json.

Pure Python 3 standard library only — no third-party dependencies.

Usage:
    python3 score.py --dataset dataset.json --run baseline/judgments
    python3 score.py --dataset dataset.json --run runs/2026-09-11/judgments \
        --map runs/2026-09-11/blind/map.json \
        --baseline baseline/baseline_metrics.json \
        --json runs/2026-09-11/metrics.json

See RUN.md in this directory for the full regression-test procedure this
script is part of, and playbooks/LEAD_QUALIFICATION_RULE.md for the rule
being scored.

--- Metric definitions (reverse-engineered and verified byte-for-byte
    against the round-11 test campaign that produced this rule; see
    baseline/baseline_metrics.json for the numbers this must reproduce) ---

For every (scenario, client) pair in dataset.json:
  - `lead_ok`  (binary):  whether the judged decision and the expected
    decision agree on whether this is a LEAD at all — i.e. whether both
    fall in {hot, warm, watch} or both fall outside it ({none, competitor}).
    This is the number the product cares about most: did we correctly
    decide to keep this person in the funnel.
  - `tier_ok` (tier): whether the judged decision exactly matches the
    expected decision (hot vs warm vs watch vs none vs competitor, all
    distinguished). Always <= lead_ok in aggregate, since an exact match
    implies a binary-lead match but not vice versa.

Aggregate metrics:
  - lead_acc  = mean(lead_ok)  over all (scenario, client) pairs
  - tier_acc  = mean(tier_ok)  over all (scenario, client) pairs
  - fit_acc   = mean(judged.fit    == expected.fit)
  - intent_acc= mean(judged.intent == expected.intent)
  - byClient / byGroup: total / lead_ok / tier_ok counts
  - groupA_warm      = recall of "warm" within group A (the "right kind of
    person, no stated need" trap group) — of the pairs EXPECTED warm in
    group A, what fraction were JUDGED warm.
  - groupE_competitor = recall of "competitor" within group E (the
    "competitor in disguise" trap group), same construction.
  - groupFG_none      = recall of "none" within groups F+G combined (the
    noise / negation trap groups), same construction.
  - confusion matrix: expected decision -> judged decision, counts.
"""

import argparse
import glob
import json
import os
import sys

CLIENT_ORDER_HINT = ["solo_agency", "ai_video", "realtor", "insurance", "mlo"]

TARGET_THRESHOLDS = {
    "lead_acc": 0.97,
    "tier_acc": 0.90,
    "groupA_warm": 0.95,
    "groupE_competitor": 0.95,
    "groupFG_none": 0.95,
}

LEAD_DECISIONS = {"hot", "warm", "watch"}
DECISIONS = ["hot", "warm", "watch", "none", "competitor"]


def norm(v):
    return (v or "").strip().lower()


def load_dataset(path):
    with open(path, "r", encoding="utf-8") as f:
        dataset = json.load(f)
    if not isinstance(dataset, list):
        raise SystemExit(f"dataset at {path} must be a JSON array of scenarios")
    scenarios = {s["id"]: s for s in dataset}
    # client set: union of expected keys, ordered by CLIENT_ORDER_HINT first
    client_set = set()
    for s in dataset:
        client_set |= set(s.get("expected", {}).keys())
    clients = [c for c in CLIENT_ORDER_HINT if c in client_set]
    clients += sorted(c for c in client_set if c not in clients)
    return dataset, scenarios, clients


def load_blind_map(path):
    if not path:
        return {}
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def _rows_from_file(fp):
    with open(fp, "r", encoding="utf-8") as f:
        data = json.load(f)
    if isinstance(data, list):
        return data
    if isinstance(data, dict):
        if isinstance(data.get("results"), list):
            return data["results"]
        if isinstance(data.get("judgments"), list):
            return data["judgments"]
    return []


def infer_client(fp, client_keys):
    """Infer which client a judgment file belongs to from its path.

    Tries, in order: the immediate parent directory name (exact match),
    then the filename stem as an exact match or as `<client>_<chunk>` /
    `<client>-<chunk>` (longest client key wins, so "ai_video_1.json"
    doesn't get clipped to a shorter key that happens to prefix-match).
    """
    parent = os.path.basename(os.path.dirname(fp))
    if parent in client_keys:
        return parent
    stem = os.path.splitext(os.path.basename(fp))[0]
    if stem in client_keys:
        return stem
    for key in sorted(client_keys, key=len, reverse=True):
        if stem.startswith(key + "_") or stem.startswith(key + "-"):
            return key
    return None


def collect_judgments(run_dir, client_keys, blind_map):
    """Returns (judgments, warnings).

    judgments: {real_scenario_id: {client: row_dict}}
    warnings: list of human-readable strings (unmatched files, id collisions)
    """
    judgments = {}
    warnings = []
    files = sorted(glob.glob(os.path.join(run_dir, "**", "*.json"), recursive=True))
    if not files:
        warnings.append(f"no .json files found under {run_dir}")
    for fp in files:
        client = infer_client(fp, client_keys)
        if client is None:
            warnings.append(f"skipped {fp}: filename doesn't match any known client "
                             f"({', '.join(sorted(client_keys))})")
            continue
        rows = _rows_from_file(fp)
        for row in rows:
            blind_id = row.get("id")
            if blind_id is None:
                warnings.append(f"skipped a row with no id in {fp}")
                continue
            real_id = blind_map.get(blind_id, blind_id)
            existing = judgments.setdefault(real_id, {})
            if client in existing:
                warnings.append(f"duplicate judgment for {real_id}/{client} "
                                 f"in {fp} (later file wins)")
            existing[client] = row
    return judgments, warnings


def compute_metrics(dataset, scenarios, clients, judgments):
    per_client = {c: {"total": 0, "lead_ok": 0, "tier_ok": 0} for c in clients}
    per_group = {}
    confusion = {}  # (exp_dec, got_dec) -> count, only where exp_dec set
    confusion_mismatch = {}  # "exp->got" -> count, only mismatches
    fit_ok = intent_ok = total = lead_ok_total = tier_ok_total = 0
    failures = []
    missing = []

    trap = {
        "groupA_warm": {"groups": {"A"}, "decision": "warm", "num": 0, "den": 0},
        "groupE_competitor": {"groups": {"E"}, "decision": "competitor", "num": 0, "den": 0},
        "groupFG_none": {"groups": {"F", "G"}, "decision": "none", "num": 0, "den": 0},
    }

    for s in dataset:
        sid = s["id"]
        group = s.get("group", "?")
        per_group.setdefault(group, {"total": 0, "lead_ok": 0, "tier_ok": 0})
        exp_all = s.get("expected", {})
        got_all = judgments.get(sid, {})
        for client in clients:
            exp = exp_all.get(client, {})
            got = got_all.get(client)
            if got is None:
                missing.append({"id": sid, "client": client, "group": group})
                got = {}
            exp_dec = norm(exp.get("decision"))
            got_dec = norm(got.get("decision"))
            exp_fit = norm(exp.get("fit"))
            got_fit = norm(got.get("fit"))
            exp_intent = norm(exp.get("intent"))
            got_intent = norm(got.get("intent"))

            is_lead_ok = (exp_dec in LEAD_DECISIONS) == (got_dec in LEAD_DECISIONS)
            is_tier_ok = bool(exp_dec) and exp_dec == got_dec

            total += 1
            per_client[client]["total"] += 1
            per_group[group]["total"] += 1
            if is_lead_ok:
                lead_ok_total += 1
                per_client[client]["lead_ok"] += 1
                per_group[group]["lead_ok"] += 1
            if is_tier_ok:
                tier_ok_total += 1
                per_client[client]["tier_ok"] += 1
                per_group[group]["tier_ok"] += 1
            if exp_fit and exp_fit == got_fit:
                fit_ok += 1
            if exp_intent == got_intent:
                intent_ok += 1

            if exp_dec:
                confusion[(exp_dec, got_dec or "(missing)")] = \
                    confusion.get((exp_dec, got_dec or "(missing)"), 0) + 1
                if exp_dec != got_dec:
                    key = f"{exp_dec}->{got_dec or '(missing)'}"
                    confusion_mismatch[key] = confusion_mismatch.get(key, 0) + 1

            for name, t in trap.items():
                if group in t["groups"] and exp_dec == t["decision"]:
                    t["den"] += 1
                    if got_dec == t["decision"]:
                        t["num"] += 1

            if not is_tier_ok:
                failures.append({
                    "id": sid, "client": client, "group": group,
                    "expected_decision": exp.get("decision"),
                    "got_decision": got.get("decision"),
                    "expected_why": exp.get("why"),
                    "got_fit_reason": got.get("fit_reason"),
                    "got_intent_reason": got.get("intent_reason"),
                    "trap": s.get("trap"),
                })

    def safe_div(n, d):
        return (n / d) if d else None

    metrics = {
        "total": total,
        "lead_acc": safe_div(lead_ok_total, total),
        "tier_acc": safe_div(tier_ok_total, total),
        "fit_acc": safe_div(fit_ok, total),
        "intent_acc": safe_div(intent_ok, total),
        "byClient": per_client,
        "byGroup": per_group,
        "confusions": confusion_mismatch,
        "confusion_matrix": {
            exp_d: {got_d: confusion.get((exp_d, got_d), 0)
                    for got_d in DECISIONS + ["(missing)"]}
            for exp_d in DECISIONS
        },
        "failures": failures,
        "failure_count": len(failures),
        "missing": missing,
        "missing_count": len(missing),
    }
    for name, t in trap.items():
        metrics[name] = safe_div(t["num"], t["den"])
        metrics[f"{name}_num"] = t["num"]
        metrics[f"{name}_den"] = t["den"]
    return metrics


def fmt_pct(v):
    return "n/a" if v is None else f"{v * 100:.2f}%"


def check_targets(metrics):
    results = {}
    for key, threshold in TARGET_THRESHOLDS.items():
        v = metrics.get(key)
        results[key] = {
            "value": v,
            "threshold": threshold,
            "pass": (v is not None and v >= threshold),
        }
    return results


def check_baseline(metrics, baseline):
    results = {}
    for key in TARGET_THRESHOLDS:
        v = metrics.get(key)
        b = baseline.get(key)
        ok = (v is not None and b is not None and v >= b)
        results[key] = {"value": v, "baseline": b, "pass": ok}
    return results


def print_report(metrics, target_results, baseline_results, warnings, show_failures):
    print("=" * 72)
    print(f"Scored {metrics['total']} (scenario x client) pairs")
    print("=" * 72)
    print()
    print(f"{'lead_acc (binary)':<24} {fmt_pct(metrics['lead_acc']):>10}   "
          f"target >= {TARGET_THRESHOLDS['lead_acc']*100:.0f}%")
    print(f"{'tier_acc (5-way)':<24} {fmt_pct(metrics['tier_acc']):>10}   "
          f"target >= {TARGET_THRESHOLDS['tier_acc']*100:.0f}%")
    print(f"{'fit_acc':<24} {fmt_pct(metrics['fit_acc']):>10}")
    print(f"{'intent_acc':<24} {fmt_pct(metrics['intent_acc']):>10}")
    print()
    print("Trap-group metrics:")
    for key in ("groupA_warm", "groupE_competitor", "groupFG_none"):
        num, den = metrics[f"{key}_num"], metrics[f"{key}_den"]
        print(f"  {key:<20} {fmt_pct(metrics[key]):>10}  ({num}/{den})   "
              f"target >= {TARGET_THRESHOLDS[key]*100:.0f}%")
    print()

    print("By client:")
    for client, st in metrics["byClient"].items():
        lp = fmt_pct(st["lead_ok"] / st["total"] if st["total"] else None)
        tp = fmt_pct(st["tier_ok"] / st["total"] if st["total"] else None)
        print(f"  {client:<14} lead {st['lead_ok']:>3}/{st['total']:<3} ({lp:>7})   "
              f"tier {st['tier_ok']:>3}/{st['total']:<3} ({tp:>7})")
    print()

    print("By group:")
    for group in sorted(metrics["byGroup"].keys()):
        st = metrics["byGroup"][group]
        lp = fmt_pct(st["lead_ok"] / st["total"] if st["total"] else None)
        tp = fmt_pct(st["tier_ok"] / st["total"] if st["total"] else None)
        print(f"  {group:<4} lead {st['lead_ok']:>3}/{st['total']:<3} ({lp:>7})   "
              f"tier {st['tier_ok']:>3}/{st['total']:<3} ({tp:>7})")
    print()

    print("Confusion matrix (expected -> judged), mismatches only:")
    if metrics["confusions"]:
        for key in sorted(metrics["confusions"], key=lambda k: -metrics["confusions"][k]):
            print(f"  {key:<24} {metrics['confusions'][key]}")
    else:
        print("  (none — every judgment matched its expected decision)")
    print()

    if metrics["missing_count"]:
        print(f"WARNING: {metrics['missing_count']} (scenario, client) pairs have no "
              f"judgment at all (counted as failures).")
        for m in metrics["missing"][:10]:
            print(f"  missing: {m['id']} / {m['client']}")
        if metrics["missing_count"] > 10:
            print(f"  ... and {metrics['missing_count'] - 10} more")
        print()

    if warnings:
        print(f"WARNINGS ({len(warnings)}):")
        for w in warnings[:20]:
            print(f"  - {w}")
        if len(warnings) > 20:
            print(f"  ... and {len(warnings) - 20} more")
        print()

    print("-" * 72)
    print("PASS/FAIL against targets:")
    all_target_pass = True
    for key, r in target_results.items():
        status = "PASS" if r["pass"] else "FAIL"
        if not r["pass"]:
            all_target_pass = False
        print(f"  [{status}] {key:<20} {fmt_pct(r['value']):>10}  (target >= "
              f"{r['threshold']*100:.0f}%)")
    print(f"  => {'ALL TARGETS MET' if all_target_pass else 'BELOW TARGET'}")
    print()

    all_baseline_pass = None
    if baseline_results is not None:
        print("PASS/FAIL against baseline (must not drop below):")
        all_baseline_pass = True
        for key, r in baseline_results.items():
            status = "PASS" if r["pass"] else "FAIL"
            if not r["pass"]:
                all_baseline_pass = False
            print(f"  [{status}] {key:<20} {fmt_pct(r['value']):>10}  (baseline "
                  f"{fmt_pct(r['baseline'])})")
        print(f"  => {'AT OR ABOVE BASELINE' if all_baseline_pass else 'REGRESSION VS BASELINE'}")
        print()

    if show_failures and metrics["failures"]:
        print("-" * 72)
        print(f"Failures ({metrics['failure_count']} total, showing up to {show_failures}):")
        for f in metrics["failures"][:show_failures]:
            print(f"  {f['id']:<6} {f['client']:<14} group {f['group']:<3}  "
                  f"expected={f['expected_decision']!r} got={f['got_decision']!r}")
            if f.get("trap"):
                print(f"         trap: {f['trap']}")
        print()

    return all_target_pass, all_baseline_pass


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                  formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--dataset", required=True, help="path to dataset.json")
    ap.add_argument("--run", required=True,
                     help="directory of judgment files (recursively scanned for *.json)")
    ap.add_argument("--map", default=None,
                     help="blind-id -> real-id map (e.g. blind/map.json). "
                          "Omit if judgment files already use real dataset ids.")
    ap.add_argument("--baseline", default=None,
                     help="baseline_metrics.json to compare against "
                          "(this run must not score below it)")
    ap.add_argument("--json", default=None,
                     help="write the full metrics dict as JSON to this path")
    ap.add_argument("--show-failures", type=int, default=15,
                     help="how many failing rows to print (default 15, 0 to hide)")
    ap.add_argument("--quiet", action="store_true",
                     help="suppress the human-readable report, print only PASS/FAIL")
    args = ap.parse_args()

    dataset, scenarios, clients = load_dataset(args.dataset)
    blind_map = load_blind_map(args.map)
    judgments, warnings = collect_judgments(args.run, set(clients), blind_map)
    metrics = compute_metrics(dataset, scenarios, clients, judgments)

    target_results = check_targets(metrics)
    baseline_results = None
    if args.baseline:
        with open(args.baseline, "r", encoding="utf-8") as f:
            baseline = json.load(f)
        baseline_results = check_baseline(metrics, baseline)

    if args.json:
        os.makedirs(os.path.dirname(os.path.abspath(args.json)), exist_ok=True)
        with open(args.json, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2, ensure_ascii=False)

    if not args.quiet:
        all_target_pass, all_baseline_pass = print_report(
            metrics, target_results, baseline_results, warnings, args.show_failures)
    else:
        all_target_pass = all(r["pass"] for r in target_results.values())
        all_baseline_pass = (all(r["pass"] for r in baseline_results.values())
                              if baseline_results is not None else None)
        binding = all_baseline_pass if all_baseline_pass is not None else all_target_pass
        print("PASS" if binding else "FAIL")

    # Exit status follows the BINDING criterion (RUN.md): when a baseline is given,
    # "not below baseline" decides; the fixed targets are informational until a rule
    # version clears them. Without a baseline the targets decide.
    ok = all_baseline_pass if all_baseline_pass is not None else all_target_pass
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
