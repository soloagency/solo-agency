#!/usr/bin/env python3
"""Deterministically score the small, domain-agnostic HOT-boundary suite.

The scorer accepts either one JSON/JSONL judgment file or a directory
containing JSON/JSONL files. JSON may be a row, an array of rows, or a
wrapper with ``results``/``judgments``. JSONL may contain rows or those
same wrappers one per line.

Binding gates are intentionally narrow and strict:

* complete, unique judgments using the canonical v12 schema/vocabulary;
* exact agreement on the fixture's structural fields;
* zero expected-non-HOT cases emitted as HOT;
* every positive flip emitted as HOT;
* every emitted HOT satisfies the full v12 conjunction and carries
  substantive fit/problem/unresolved/active/acquisition evidence; and
* metamorphic variants retain the same structural judgment.

Usage (from this directory):

    python3 score_hot_boundary.py \
      --cases hot_boundary_cases.json \
      --expected hot_boundary_expected.json \
      --judgments runs/<date>/hot_boundary/judgments.jsonl \
      --json runs/<date>/hot_boundary/metrics.json

For the required five-record extractor canary, add:

    --ids HB001,HB002,HB003,HB022,HB024
"""

import argparse
import glob
import json
import os
import re
import sys
from collections import Counter, defaultdict


HOT_REQUIRED_VALUES = {
    "fit": "high",
    "problem_relevance": "proven",
    "problem_state": "unresolved",
    "resolution_activity": "active",
    "counterfactual_result": "pass",
    "lead_level": "hot",
    "decision": "hot",
}
HOT_ALLOWED_VALUES = {
    "acquisition_posture": {"explicit", "open"},
    "intent": {"explicit", "implied"},
}
HOT_INTENT_POSTURE = {"explicit": "explicit", "implied": "open"}
HOT_EVIDENCE_FIELDS = (
    "fit_evidence",
    "problem_outcome",
    "problem_evidence",
    "unresolved_evidence",
    "active_resolution_evidence",
    "acquisition_evidence",
)
REQUIRED_REASON_FIELDS = ("fit_reason", "intent_reason")
PLACEHOLDER_EVIDENCE = {
    "",
    "-",
    "n/a",
    "na",
    "none",
    "no evidence",
    "not applicable",
    "not shown",
    "not_shown",
    "unknown",
}


def norm(value):
    return str(value).strip().lower() if value is not None else ""


def safe_div(num, den):
    return (num / den) if den else None


def fmt_pct(value):
    return "n/a" if value is None else f"{value * 100:.1f}%"


def substantive_evidence(value):
    text = norm(value)
    return (
        text not in PLACEHOLDER_EVIDENCE
        and len(text) >= 4
        and bool(re.search(r"[a-z0-9\u00c0-\u024f\u1e00-\u1eff]", text))
    )


def _unwrap_json(value):
    if isinstance(value, list):
        return value
    if isinstance(value, dict):
        for key in ("results", "judgments"):
            if isinstance(value.get(key), list):
                return value[key]
        if "id" in value:
            return [value]
    return []


def load_judgment_rows(path):
    """Return (rows, parse_errors) from a JSON/JSONL file or directory."""
    if os.path.isdir(path):
        files = sorted(
            glob.glob(os.path.join(path, "**", "*.json"), recursive=True)
            + glob.glob(os.path.join(path, "**", "*.jsonl"), recursive=True)
        )
    elif os.path.isfile(path):
        files = [path]
    else:
        return [], [f"judgment path does not exist: {path}"]

    rows = []
    errors = []
    if not files:
        errors.append(f"no .json or .jsonl files found under {path}")
    for file_path in files:
        if file_path.lower().endswith(".jsonl"):
            try:
                with open(file_path, "r", encoding="utf-8") as handle:
                    for line_no, line in enumerate(handle, 1):
                        if not line.strip():
                            continue
                        try:
                            value = json.loads(line)
                        except json.JSONDecodeError as exc:
                            errors.append(f"{file_path}:{line_no}: invalid JSON: {exc}")
                            continue
                        unwrapped = _unwrap_json(value)
                        if not unwrapped:
                            errors.append(
                                f"{file_path}:{line_no}: expected a judgment row or results/judgments array"
                            )
                        rows.extend(unwrapped)
            except OSError as exc:
                errors.append(f"{file_path}: could not read JSONL: {exc}")
        else:
            try:
                with open(file_path, "r", encoding="utf-8") as handle:
                    value = json.load(handle)
            except (OSError, json.JSONDecodeError) as exc:
                errors.append(f"{file_path}: could not parse JSON: {exc}")
                continue
            unwrapped = _unwrap_json(value)
            if not unwrapped:
                errors.append(
                    f"{file_path}: expected a judgment row, array, or results/judgments array"
                )
            rows.extend(unwrapped)
    return rows, errors


def load_contract(cases_path, expected_path):
    with open(cases_path, "r", encoding="utf-8") as handle:
        cases_doc = json.load(handle)
    with open(expected_path, "r", encoding="utf-8") as handle:
        expected_doc = json.load(handle)

    case_rows = cases_doc.get("cases") if isinstance(cases_doc, dict) else cases_doc
    if not isinstance(case_rows, list):
        raise SystemExit(f"{cases_path}: cases must be a JSON array or object with cases[]")
    cases = {}
    for row in case_rows:
        if not isinstance(row, dict) or not row.get("id"):
            raise SystemExit(f"{cases_path}: every case must be an object with id")
        if row["id"] in cases:
            raise SystemExit(f"{cases_path}: duplicate case id {row['id']}")
        if not isinstance(row.get("client_offer"), dict) or not isinstance(row.get("item"), dict):
            raise SystemExit(f"{cases_path}: {row['id']} must embed client_offer and item objects")
        cases[row["id"]] = row

    expected = expected_doc.get("cases")
    if not isinstance(expected, dict):
        raise SystemExit(f"{expected_path}: cases must be an object keyed by case id")
    if set(cases) != set(expected):
        missing_labels = sorted(set(cases) - set(expected))
        missing_cases = sorted(set(expected) - set(cases))
        raise SystemExit(
            f"case/label id mismatch; unlabeled cases={missing_labels}, labels without cases={missing_cases}"
        )

    vocabulary = expected_doc.get("closed_vocabulary", {})
    binding_fields = expected_doc.get("binding_fields", [])
    for case_id, label in expected.items():
        exp = label.get("expected")
        if not isinstance(exp, dict):
            raise SystemExit(f"{expected_path}: {case_id} must contain expected object")
        for field in binding_fields:
            if field not in exp:
                raise SystemExit(f"{expected_path}: {case_id} lacks binding expected field {field}")
        for field, value in exp.items():
            if field in vocabulary and norm(value) not in set(vocabulary[field]):
                raise SystemExit(
                    f"{expected_path}: {case_id} has invalid expected {field}={value!r}"
                )
        hot_expected = label.get("hot_expected")
        if not isinstance(hot_expected, bool):
            raise SystemExit(f"{expected_path}: {case_id} hot_expected must be boolean")
        if hot_expected != (norm(exp.get("decision")) == "hot"):
            raise SystemExit(
                f"{expected_path}: {case_id} hot_expected disagrees with expected decision"
            )
        if hot_expected:
            for field, required in HOT_REQUIRED_VALUES.items():
                if norm(exp.get(field)) != required:
                    raise SystemExit(
                        f"{expected_path}: HOT label {case_id} requires {field}={required}"
                    )
            for field, allowed in HOT_ALLOWED_VALUES.items():
                if norm(exp.get(field)) not in allowed:
                    raise SystemExit(
                        f"{expected_path}: HOT label {case_id} requires {field} in {sorted(allowed)}"
                    )
            expected_posture = HOT_INTENT_POSTURE[norm(exp.get("intent"))]
            if norm(exp.get("acquisition_posture")) != expected_posture:
                raise SystemExit(
                    f"{expected_path}: HOT label {case_id} with intent={exp.get('intent')!r} "
                    f"requires acquisition_posture={expected_posture}"
                )

    metamorphic_fields = expected_doc.get("metamorphic_fields", binding_fields)
    for pair in expected_doc.get("metamorphic_sets", []):
        pair_ids = pair.get("case_ids", [])
        if len(pair_ids) < 2 or any(case_id not in expected for case_id in pair_ids):
            raise SystemExit(f"{expected_path}: invalid metamorphic set {pair.get('id')!r}")
        for field in metamorphic_fields:
            values = {norm(expected[case_id]["expected"].get(field)) for case_id in pair_ids}
            if len(values) != 1:
                raise SystemExit(
                    f"{expected_path}: metamorphic set {pair.get('id')!r} disagrees on {field}"
                )
    for pair in expected_doc.get("flip_pairs", []):
        negative_id = pair.get("nonhot_id")
        hot_id = pair.get("hot_id")
        if negative_id not in expected or hot_id not in expected:
            raise SystemExit(f"{expected_path}: flip pair references unknown id: {pair}")
        if expected[negative_id]["hot_expected"] or not expected[hot_id]["hot_expected"]:
            raise SystemExit(f"{expected_path}: flip pair direction is invalid: {pair}")
    return cases, expected_doc


def select_contract(cases, expected_doc, selected_ids):
    """Restrict a contract to a canary subset without weakening its gates.

    Pair-level checks run only when every member of that metamorphic/flip pair
    is in the selected subset. Row-level schema, evidence and expected-label
    gates remain unchanged.
    """
    selected = set(selected_ids)
    unknown = sorted(selected - set(cases))
    if unknown:
        raise SystemExit(f"--ids contains unknown case ids: {', '.join(unknown)}")
    if not selected:
        raise SystemExit("--ids selected no cases")
    filtered_cases = {case_id: cases[case_id] for case_id in cases if case_id in selected}
    filtered_doc = dict(expected_doc)
    filtered_doc["cases"] = {
        case_id: expected_doc["cases"][case_id]
        for case_id in expected_doc["cases"]
        if case_id in selected
    }
    filtered_doc["metamorphic_sets"] = [
        pair for pair in expected_doc.get("metamorphic_sets", [])
        if set(pair.get("case_ids", [])) <= selected
    ]
    filtered_doc["flip_pairs"] = [
        pair for pair in expected_doc.get("flip_pairs", [])
        if {pair.get("nonhot_id"), pair.get("hot_id")} <= selected
    ]
    return filtered_cases, filtered_doc


def collect_judgments(rows, known_ids):
    judgments = {}
    errors = []
    unknown = []
    duplicates = []
    for index, row in enumerate(rows, 1):
        if not isinstance(row, dict):
            errors.append(f"judgment row {index} is not an object")
            continue
        case_id = row.get("id")
        if not case_id:
            errors.append(f"judgment row {index} has no id")
            continue
        if case_id not in known_ids:
            unknown.append(case_id)
            continue
        if case_id in judgments:
            duplicates.append(case_id)
            continue
        judgments[case_id] = row
    return judgments, errors, sorted(set(unknown)), sorted(set(duplicates))


def score(cases, expected_doc, judgments, parse_errors, row_errors, unknown, duplicates):
    expected_cases = expected_doc["cases"]
    required_fields = expected_doc["required_output_fields"]
    binding_fields = expected_doc["binding_fields"]
    info_fields = expected_doc.get("informational_fields", [])
    vocabulary = expected_doc["closed_vocabulary"]
    required_version = norm(expected_doc.get("rule_version"))

    missing_ids = sorted(set(cases) - set(judgments))
    schema_errors = list(parse_errors) + list(row_errors)
    field_failures = []
    info_field_failures = []
    evidence_errors = []
    hot_gate_errors = []
    exact_ok = 0
    expected_hot = 0
    judged_hot = 0
    hot_true_positive = 0
    false_hot = []
    false_hot_by_family = Counter()
    false_hot_by_expected_decision = Counter()
    missed_hot = []

    for case_id in sorted(cases):
        label = expected_cases[case_id]
        exp = label["expected"]
        exp_is_hot = bool(label.get("hot_expected", exp.get("decision") == "hot"))
        if exp_is_hot:
            expected_hot += 1

        row = judgments.get(case_id)
        if row is None:
            continue

        for field in required_fields:
            if field not in row:
                schema_errors.append(f"{case_id}: missing required field {field}")
        if norm(row.get("lead_rule_version")) != required_version:
            schema_errors.append(
                f"{case_id}: lead_rule_version={row.get('lead_rule_version')!r}; expected {expected_doc.get('rule_version')!r}"
            )
        for field, allowed in vocabulary.items():
            value = norm(row.get(field))
            if value not in set(allowed):
                schema_errors.append(
                    f"{case_id}: {field}={row.get(field)!r}; allowed={','.join(allowed)}"
                )
        for field in REQUIRED_REASON_FIELDS:
            if not substantive_evidence(row.get(field)):
                evidence_errors.append({
                    "id": case_id,
                    "field": field,
                    "reason": "required reasoning field is empty or a placeholder",
                    "value": row.get(field),
                })

        decision = norm(row.get("decision"))
        lead_level = norm(row.get("lead_level"))
        expected_lead_level = decision if decision in {"hot", "warm", "watch"} else "none"
        if decision in set(vocabulary.get("decision", [])) and lead_level != expected_lead_level:
            schema_errors.append(
                f"{case_id}: decision={decision} requires lead_level={expected_lead_level}, got {lead_level or '(missing)'}"
            )

        predicted_hot = decision == "hot" or lead_level == "hot"
        if decision == "hot":
            judged_hot += 1
        if exp_is_hot and decision == "hot":
            hot_true_positive += 1
        elif exp_is_hot:
            missed_hot.append({
                "id": case_id,
                "expected": "hot",
                "got": decision or "(missing)",
                "family": label.get("family"),
            })
        if not exp_is_hot and predicted_hot:
            false_hot.append({
                "id": case_id,
                "expected": exp.get("decision"),
                "got_decision": decision or "(missing)",
                "got_lead_level": lead_level or "(missing)",
                "family": label.get("family"),
                "invariant": label.get("invariant"),
            })
            false_hot_by_family[label.get("family", "?")] += 1
            false_hot_by_expected_decision[exp.get("decision", "?")] += 1

        case_all_exact = True
        for field in binding_fields:
            got_value = norm(row.get(field))
            exp_value = norm(exp.get(field))
            if got_value != exp_value:
                case_all_exact = False
                field_failures.append({
                    "id": case_id,
                    "field": field,
                    "expected": exp_value,
                    "got": got_value or "(missing)",
                    "family": label.get("family"),
                })
        if case_all_exact:
            exact_ok += 1
        for field in info_fields:
            if field in exp and norm(row.get(field)) != norm(exp.get(field)):
                info_field_failures.append({
                    "id": case_id,
                    "field": field,
                    "expected": norm(exp.get(field)),
                    "got": norm(row.get(field)) or "(missing)",
                })

        # Conditional evidence contract for every row. Negative states may use
        # an empty/not-shown evidence string, but every positive state must be
        # supported by a substantive item-specific evidence field.
        evidence_requirements = []
        if norm(row.get("fit")) in {"high", "medium"}:
            evidence_requirements.append(("fit_evidence", "fit is high/medium"))
        if norm(row.get("problem_relevance")) in {"proven", "plausible"}:
            evidence_requirements.append(("problem_evidence", "problem relevance is positive"))
        if norm(row.get("problem_state")) == "unresolved":
            evidence_requirements.append(("unresolved_evidence", "problem is unresolved"))
        if norm(row.get("resolution_activity")) == "active":
            evidence_requirements.append(("active_resolution_evidence", "resolution activity is active"))
        if norm(row.get("acquisition_posture")) in {"explicit", "open"}:
            evidence_requirements.append(("acquisition_evidence", "acquisition posture is explicit/open"))
        if norm(row.get("urgency")) in {"immediate", "soon"}:
            evidence_requirements.append(("timing_evidence", "urgency is immediate/soon"))
        for evidence_field, reason in evidence_requirements:
            if not substantive_evidence(row.get(evidence_field)):
                evidence_errors.append({
                    "id": case_id,
                    "field": evidence_field,
                    "reason": reason,
                    "value": row.get(evidence_field),
                })

        if predicted_hot:
            for field, required in HOT_REQUIRED_VALUES.items():
                if norm(row.get(field)) != required:
                    hot_gate_errors.append(
                        f"{case_id}: HOT requires {field}={required}, got {norm(row.get(field)) or '(missing)'}"
                    )
            for field, allowed in HOT_ALLOWED_VALUES.items():
                if norm(row.get(field)) not in allowed:
                    hot_gate_errors.append(
                        f"{case_id}: HOT requires {field} in {sorted(allowed)}, got {norm(row.get(field)) or '(missing)'}"
                    )
            expected_posture = HOT_INTENT_POSTURE.get(norm(row.get("intent")))
            if expected_posture and norm(row.get("acquisition_posture")) != expected_posture:
                hot_gate_errors.append(
                    f"{case_id}: HOT intent={norm(row.get('intent'))} requires "
                    f"acquisition_posture={expected_posture}"
                )
            for field in HOT_EVIDENCE_FIELDS:
                if not substantive_evidence(row.get(field)):
                    hot_gate_errors.append(f"{case_id}: HOT lacks substantive {field}")
            evidence_uses = defaultdict(list)
            for field in HOT_EVIDENCE_FIELDS + ("timing_evidence",):
                if substantive_evidence(row.get(field)):
                    evidence_uses[norm(row.get(field))].append(field)
            for fields in evidence_uses.values():
                if len(fields) > 1:
                    hot_gate_errors.append(
                        f"{case_id}: HOT reuses one generic evidence string across {','.join(fields)}"
                    )

    metamorphic_failures = []
    metamorphic_fields = expected_doc.get("metamorphic_fields", binding_fields)
    for pair in expected_doc.get("metamorphic_sets", []):
        pair_rows = [(case_id, judgments.get(case_id)) for case_id in pair["case_ids"]]
        if any(row is None for _, row in pair_rows):
            metamorphic_failures.append({
                "set": pair["id"],
                "field": "coverage",
                "values": {case_id: "(missing)" if row is None else "present" for case_id, row in pair_rows},
            })
            continue
        for field in metamorphic_fields:
            values = {case_id: norm(row.get(field)) for case_id, row in pair_rows}
            if len(set(values.values())) != 1:
                metamorphic_failures.append({
                    "set": pair["id"],
                    "field": field,
                    "values": values,
                })

    flip_failures = []
    for pair in expected_doc.get("flip_pairs", []):
        negative = judgments.get(pair["nonhot_id"])
        positive = judgments.get(pair["hot_id"])
        neg_hot = negative is not None and (
            norm(negative.get("decision")) == "hot" or norm(negative.get("lead_level")) == "hot"
        )
        pos_hot = positive is not None and norm(positive.get("decision")) == "hot"
        if neg_hot or not pos_hot:
            flip_failures.append({
                "nonhot_id": pair["nonhot_id"],
                "hot_id": pair["hot_id"],
                "nonhot_got": norm(negative.get("decision")) if negative else "(missing)",
                "hot_got": norm(positive.get("decision")) if positive else "(missing)",
                "semantic_change": pair.get("only_semantic_change"),
            })

    total = len(cases)
    expected_nonhot = total - expected_hot
    exact_accuracy = safe_div(exact_ok, total)
    hot_precision = safe_div(hot_true_positive, judged_hot)
    hot_recall = safe_div(hot_true_positive, expected_hot)
    false_hot_rate = safe_div(len(false_hot), expected_nonhot)

    binding_pass = not any((
        missing_ids,
        unknown,
        duplicates,
        schema_errors,
        field_failures,
        evidence_errors,
        hot_gate_errors,
        false_hot,
        missed_hot,
        metamorphic_failures,
        flip_failures,
    ))

    return {
        "total": total,
        "judged": len(judgments),
        "missing_ids": missing_ids,
        "unknown_ids": unknown,
        "duplicate_ids": duplicates,
        "schema_errors": schema_errors,
        "field_failures": field_failures,
        "informational_field_failures": info_field_failures,
        "evidence_errors": evidence_errors,
        "hot_gate_errors": hot_gate_errors,
        "metamorphic_failures": metamorphic_failures,
        "flip_failures": flip_failures,
        "exact_structural_cases": exact_ok,
        "exact_structural_accuracy": exact_accuracy,
        "expected_hot": expected_hot,
        "expected_nonhot": expected_nonhot,
        "judged_hot": judged_hot,
        "hot_true_positive": hot_true_positive,
        "hot_precision": hot_precision,
        "hot_recall": hot_recall,
        "nonhot_to_hot_total": len(false_hot),
        "nonhot_to_hot_rate": false_hot_rate,
        "nonhot_to_hot_breakdown": {
            "by_family": dict(sorted(false_hot_by_family.items())),
            "by_expected_decision": dict(sorted(false_hot_by_expected_decision.items())),
            "cases": false_hot,
        },
        "missed_hot": missed_hot,
        "binding_pass": binding_pass,
    }


def print_report(metrics, show):
    print("=" * 74)
    print("HOT-boundary canary (v12, domain-agnostic)")
    print("=" * 74)
    print(
        f"coverage             {metrics['judged']}/{metrics['total']} "
        f"(missing {len(metrics['missing_ids'])}, unknown {len(metrics['unknown_ids'])}, "
        f"duplicates {len(metrics['duplicate_ids'])})"
    )
    print(
        f"structural exact     {metrics['exact_structural_cases']}/{metrics['total']} "
        f"({fmt_pct(metrics['exact_structural_accuracy'])})"
    )
    print(
        f"HOT precision        {fmt_pct(metrics['hot_precision'])} "
        f"({metrics['hot_true_positive']}/{metrics['judged_hot']} judged HOT)"
    )
    print(
        f"HOT recall           {fmt_pct(metrics['hot_recall'])} "
        f"({metrics['hot_true_positive']}/{metrics['expected_hot']} expected HOT)"
    )
    print(
        f"non-HOT -> HOT       {metrics['nonhot_to_hot_total']}/{metrics['expected_nonhot']} "
        f"({fmt_pct(metrics['nonhot_to_hot_rate'])})  BINDING: must be zero"
    )
    print(
        f"contract errors      schema={len(metrics['schema_errors'])} "
        f"evidence={len(metrics['evidence_errors'])} hot_gate={len(metrics['hot_gate_errors'])}"
    )
    print(
        f"pair errors          metamorphic={len(metrics['metamorphic_failures'])} "
        f"positive_flip={len(metrics['flip_failures'])}"
    )
    if metrics["nonhot_to_hot_breakdown"]["by_family"]:
        print("false-HOT families:  " + json.dumps(
            metrics["nonhot_to_hot_breakdown"]["by_family"], ensure_ascii=False, sort_keys=True
        ))
    print("-" * 74)
    print("BINDING PASS" if metrics["binding_pass"] else "BINDING FAIL")

    groups = (
        ("missing ids", metrics["missing_ids"]),
        ("unknown ids", metrics["unknown_ids"]),
        ("duplicate ids", metrics["duplicate_ids"]),
        ("schema errors", metrics["schema_errors"]),
        ("structural field mismatches", metrics["field_failures"]),
        ("evidence errors", metrics["evidence_errors"]),
        ("HOT gate errors", metrics["hot_gate_errors"]),
        ("non-HOT -> HOT", metrics["nonhot_to_hot_breakdown"]["cases"]),
        ("missed HOT", metrics["missed_hot"]),
        ("metamorphic failures", metrics["metamorphic_failures"]),
        ("positive-flip failures", metrics["flip_failures"]),
    )
    remaining = show
    if show:
        for title, values in groups:
            if not values or remaining <= 0:
                continue
            print()
            print(f"{title} ({len(values)}):")
            for value in values[:remaining]:
                print("  " + (value if isinstance(value, str) else json.dumps(value, ensure_ascii=False)))
            remaining -= min(len(values), remaining)


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--cases", default="hot_boundary_cases.json", help="blind case fixture JSON")
    parser.add_argument("--expected", default="hot_boundary_expected.json", help="expected labels/contract JSON")
    parser.add_argument("--judgments", required=True, help="judgment JSON/JSONL file or directory")
    parser.add_argument(
        "--ids",
        default=None,
        help="optional comma-separated case ids for a small canary subset",
    )
    parser.add_argument("--json", default=None, help="optional metrics JSON output path")
    parser.add_argument("--show-failures", type=int, default=20, help="maximum failure rows to print")
    parser.add_argument("--quiet", action="store_true", help="print only PASS or FAIL")
    args = parser.parse_args()

    cases, expected_doc = load_contract(args.cases, args.expected)
    if args.ids:
        selected_ids = [part.strip() for part in args.ids.split(",") if part.strip()]
        cases, expected_doc = select_contract(cases, expected_doc, selected_ids)
    rows, parse_errors = load_judgment_rows(args.judgments)
    judgments, row_errors, unknown, duplicates = collect_judgments(rows, set(cases))
    metrics = score(cases, expected_doc, judgments, parse_errors, row_errors, unknown, duplicates)

    if args.json:
        os.makedirs(os.path.dirname(os.path.abspath(args.json)), exist_ok=True)
        with open(args.json, "w", encoding="utf-8") as handle:
            json.dump(metrics, handle, indent=2, ensure_ascii=False)

    if args.quiet:
        print("PASS" if metrics["binding_pass"] else "FAIL")
    else:
        print_report(metrics, args.show_failures)
    return 0 if metrics["binding_pass"] else 1


if __name__ == "__main__":
    sys.exit(main())
