#!/usr/bin/env python3
"""Build a self-contained HTML report for one lead-qualification-rule run.

Adapted from the throwaway scratch script that produced the round-11 report
during the original 11-round campaign (soloagency_leadtest_2026-09-11/
build_report_v4.py) — rewritten to take explicit arguments instead of
hardcoded scratch paths, and to compute its own metrics (via score.py in
this directory) instead of reading pre-computed scratch files.

Usage:
    python3 build_report.py --dataset dataset.json --run baseline/judgments \
        --clients clients.json --rule ../../LEAD_QUALIFICATION_RULE.md \
        --out /path/to/report.html
    # add --map runs/<date>/blind/map.json when judgments use blind ids
    # add --baseline baseline/baseline_metrics.json to show baseline deltas
"""

import argparse
import html
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import score  # noqa: E402  (local module, see score.py in this directory)

CLIENT_NAME_OVERRIDES = {
    "solo_agency": "Solo Agency",
    "ai_video": "AI Video",
    "realtor": "Realtor",
    "insurance": "Insurance",
    "mlo": "MLO",
}

DECISION_CLASS = {
    "hot": "dec-hot", "warm": "dec-warm", "watch": "dec-watch",
    "none": "dec-none", "competitor": "dec-competitor",
}
DECISION_LABEL = {
    "hot": "Hot", "warm": "Warm", "watch": "Watch",
    "none": "None", "competitor": "Competitor",
}


def esc(s):
    return html.escape(str(s) if s is not None else "", quote=True)


def dec_badge(dec):
    d = score.norm(dec)
    cls = DECISION_CLASS.get(d, "dec-none")
    label = DECISION_LABEL.get(d, dec or "—")
    return f'<span class="dec-badge {cls}">{esc(label)}</span>'


def client_label(c, clients_meta):
    if c in CLIENT_NAME_OVERRIDES:
        return CLIENT_NAME_OVERRIDES[c]
    for entry in clients_meta.get("clients", []):
        if entry.get("key") == c:
            return entry.get("name", c)
    return c


def client_profile_block(c):
    name = c.get("name", c.get("key", ""))
    types_html = "".join(f"<li>{esc(t)}</li>" for t in c.get("types", []))
    return f"""
    <div class="profile-card">
      <h3>{esc(name)}</h3>
      <p><b>Sells:</b> {esc(c.get('sells', ''))}</p>
      <p><b>Sells to:</b> {esc(c.get('sells_to', ''))}</p>
      <p><b>Types:</b></p>
      <ul>{types_html}</ul>
      <p><b>Why they need it:</b> {esc(c.get('why_they_need', ''))}</p>
      <p><b>Location:</b> {esc(c.get('location', ''))}</p>
      <p><b>Competitors:</b> {esc(c.get('competitors', ''))}</p>
      <p><b>Not buyers:</b> {esc(c.get('not_buyers', ''))}</p>
    </div>"""


def build_row(sid, client, scen, exp, got, ok, group, client_names):
    author_industry = scen.get("author_industry", "")
    person_type = got.get("person_type", "")
    sells_match = got.get("sells_to_match", "")
    surface = scen.get("surface", "")
    community = scen.get("community", "")
    author_line = scen.get("author_line", "")
    text = scen.get("text", "")
    exp_fit, exp_intent, exp_dec, exp_why = (
        exp.get("fit", ""), exp.get("intent", ""), exp.get("decision", ""), exp.get("why", ""))
    got_fit, got_intent, got_dec = got.get("fit", ""), got.get("intent", ""), got.get("decision", "")
    fit_reason, intent_reason = got.get("fit_reason", ""), got.get("intent_reason", "")
    ok_class = "row-ok" if ok else "row-bad"
    ok_label = "OK" if ok else "WRONG"
    search_blob = " ".join([
        sid, group, client, author_industry, person_type, sells_match,
        surface, community, author_line, text, exp_fit, exp_intent, exp_dec, exp_why,
        got_fit, got_intent, got_dec, fit_reason, intent_reason,
    ]).lower()
    content_html = ""
    if author_line:
        content_html += f'<div class="author-line">{esc(author_line)}</div>'
    content_html += f'<div class="content-text">{esc(text)}</div>'
    return f"""
    <tr class="data-row {ok_class}" data-client="{esc(client)}" data-group="{esc(group)}" data-ok="{'1' if ok else '0'}" data-search="{esc(search_blob)}">
      <td class="col-id">{esc(sid)}</td>
      <td class="col-group">{esc(group)}</td>
      <td class="col-industry"><div><b>{esc(author_industry)}</b></div><div class="muted">{esc(person_type)}</div><div class="muted small">{esc(sells_match)}</div></td>
      <td class="col-surface"><div>{esc(surface)}</div><div class="muted">{esc(community)}</div></td>
      <td class="col-content">{content_html}</td>
      <td class="col-client">{esc(client_names.get(client, client))}</td>
      <td class="col-expected"><div>fit: <b>{esc(exp_fit)}</b> &middot; intent: <b>{esc(exp_intent)}</b></div><div>{dec_badge(exp_dec)}</div><div class="muted small">{esc(exp_why)}</div></td>
      <td class="col-judged"><div>fit: <b>{esc(got_fit)}</b> &middot; intent: <b>{esc(got_intent)}</b></div><div>{dec_badge(got_dec)}</div></td>
      <td class="col-reason"><div class="muted small">{esc(fit_reason)}</div><div class="muted small">{esc(intent_reason)}</div></td>
      <td class="col-ok"><span class="ok-badge {'ok-yes' if ok else 'ok-no'}">{ok_label}</span></td>
    </tr>"""


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                  formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--dataset", required=True)
    ap.add_argument("--run", required=True, help="directory of judgment files")
    ap.add_argument("--clients", required=True, help="clients.json (buyer profiles)")
    ap.add_argument("--rule", required=True, help="path to the rule .md file")
    ap.add_argument("--out", required=True, help="output .html path")
    ap.add_argument("--map", default=None, help="blind id -> real id map")
    ap.add_argument("--baseline", default=None, help="baseline_metrics.json for delta display")
    args = ap.parse_args()

    dataset, scenarios, clients = score.load_dataset(args.dataset)
    blind_map = score.load_blind_map(args.map)
    judgments, warnings = score.collect_judgments(args.run, set(clients), blind_map)
    metrics = score.compute_metrics(dataset, scenarios, clients, judgments)
    target_results = score.check_targets(metrics)

    baseline = None
    baseline_results = None
    if args.baseline:
        with open(args.baseline, "r", encoding="utf-8") as f:
            baseline = json.load(f)
        baseline_results = score.check_baseline(metrics, baseline)

    with open(args.clients, "r", encoding="utf-8") as f:
        clients_meta = json.load(f)
    with open(args.rule, "r", encoding="utf-8") as f:
        rule_text = f.read()

    client_names = {c: client_label(c, clients_meta) for c in clients}

    # ---- rows ----
    rows_data = []
    for s in dataset:
        sid = s["id"]
        group = s.get("group", "?")
        exp_all = s.get("expected", {})
        got_all = judgments.get(sid, {})
        for client in clients:
            exp = exp_all.get(client, {})
            got = got_all.get(client, {})
            ok = score.norm(exp.get("decision")) == score.norm(got.get("decision")) and exp.get("decision")
            rows_data.append((sid, client, s, exp, got, bool(ok), group))
    rows_data.sort(key=lambda r: (r[0], clients.index(r[1])))
    main_rows_html = "\n".join(
        build_row(sid, client, scen, exp, got, ok, group, client_names)
        for sid, client, scen, exp, got, ok, group in rows_data)
    total_rows = len(rows_data)

    # ---- client / group accuracy tables (tier = exact match, shown as the headline "accuracy") ----
    client_acc_rows = "\n".join(
        f"""<tr><td>{esc(client_names[c])}</td>
             <td>{st['lead_ok']}/{st['total']}</td><td>{(st['lead_ok']/st['total']*100 if st['total'] else 0):.1f}%</td>
             <td>{st['tier_ok']}/{st['total']}</td><td>{(st['tier_ok']/st['total']*100 if st['total'] else 0):.1f}%</td></tr>"""
        for c, st in metrics["byClient"].items())
    group_acc_rows = "\n".join(
        f"""<tr><td>{esc(g)}</td>
             <td>{st['lead_ok']}/{st['total']}</td><td>{(st['lead_ok']/st['total']*100 if st['total'] else 0):.1f}%</td>
             <td>{st['tier_ok']}/{st['total']}</td><td>{(st['tier_ok']/st['total']*100 if st['total'] else 0):.1f}%</td></tr>"""
        for g, st in sorted(metrics["byGroup"].items()))

    # ---- confusion matrix ----
    decs = score.DECISIONS + ["(missing)"]
    conf_header = "".join(f"<th>{DECISION_LABEL.get(d, d)}</th>" for d in decs)
    conf_rows = []
    for exp_d in score.DECISIONS:
        cells = []
        for got_d in decs:
            c = metrics["confusion_matrix"].get(exp_d, {}).get(got_d, 0)
            cls = ""
            if c and exp_d == got_d:
                cls = ' class="conf-diag"'
            elif c:
                cls = ' class="conf-off"'
            cells.append(f"<td{cls}>{c if c else '&middot;'}</td>")
        conf_rows.append(f"<tr><th>{DECISION_LABEL.get(exp_d, exp_d)}</th>{''.join(cells)}</tr>")
    confusion_table = "\n".join(conf_rows)

    # ---- target / baseline badges ----
    def target_rows():
        out = []
        for key, r in target_results.items():
            status = "ok-yes" if r["pass"] else "ok-no"
            out.append(f"<tr><td>{key}</td><td>{score.fmt_pct(r['value'])}</td>"
                        f"<td>&ge; {r['threshold']*100:.0f}%</td>"
                        f"<td><span class='ok-badge {status}'>{'PASS' if r['pass'] else 'FAIL'}</span></td></tr>")
        return "\n".join(out)

    baseline_section = ""
    if baseline_results is not None:
        brows = []
        for key, r in baseline_results.items():
            status = "ok-yes" if r["pass"] else "ok-no"
            brows.append(f"<tr><td>{key}</td><td>{score.fmt_pct(r['value'])}</td>"
                          f"<td>{score.fmt_pct(r['baseline'])}</td>"
                          f"<td><span class='ok-badge {status}'>{'PASS' if r['pass'] else 'FAIL'}</span></td></tr>")
        baseline_section = f"""
  <h2>Vs. baseline</h2>
  <div class="table-scroll">
  <table class="metrics-table">
    <thead><tr><th>Metric</th><th>This run</th><th>Baseline</th><th>Result</th></tr></thead>
    <tbody>{''.join(brows)}</tbody>
  </table>
  </div>"""

    profiles_html = "".join(client_profile_block(c) for c in clients_meta.get("clients", []))

    all_target_pass = all(r["pass"] for r in target_results.values())
    all_baseline_pass = (all(r["pass"] for r in baseline_results.values())
                          if baseline_results is not None else None)

    missing_note = ""
    if metrics["missing_count"]:
        missing_note = (f'<div class="note">Warning: {metrics["missing_count"]} '
                         f'(scenario, client) pairs had no judgment at all in this run '
                         f'and were scored as failures.</div>')
    warnings_note = ""
    if warnings:
        warnings_html = "".join(f"<li>{esc(w)}</li>" for w in warnings)
        warnings_note = f'<div class="note">Loader warnings:<ul>{warnings_html}</ul></div>'

    html_out = f"""<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Lead rule regression report</title>
<style>
:root {{
  --bg: #faf9f7; --panel: #ffffff; --border: #e2ddd5; --text: #1f1c18; --muted: #75706a;
  --accent: #b5432f; --accent-soft: #f4e2dd;
  --hot: #c0392b; --warm: #d98324; --watch: #c9a227; --none: #8a8680; --competitor: #7b4fa3;
  --ok: #1f7a3d; --bad: #b23b2e;
  --row-bad-bg: #fdeeec; --row-ok-bg: transparent;
}}
@media (prefers-color-scheme: dark) {{
  :root:not([data-theme="light"]) {{
    --bg: #17140f; --panel: #211d18; --border: #3a352d; --text: #f1ede6; --muted: #a89f92;
    --accent: #e27a5f; --accent-soft: #3a2620;
    --hot: #e0564a; --warm: #e6a13c; --watch: #d9bf5a; --none: #9a958c; --competitor: #a582d1;
    --ok: #4caf6f; --bad: #e0716a;
    --row-bad-bg: #34211d; --row-ok-bg: transparent;
  }}
}}
:root[data-theme="dark"] {{
  --bg: #17140f; --panel: #211d18; --border: #3a352d; --text: #f1ede6; --muted: #a89f92;
  --accent: #e27a5f; --accent-soft: #3a2620;
  --hot: #e0564a; --warm: #e6a13c; --watch: #d9bf5a; --none: #9a958c; --competitor: #a582d1;
  --ok: #4caf6f; --bad: #e0716a;
  --row-bad-bg: #34211d; --row-ok-bg: transparent;
}}
* {{ box-sizing: border-box; }}
body {{ background: var(--bg); color: var(--text); font-family: -apple-system, "Segoe UI", Roboto, sans-serif; }}
.wrap {{ max-width: 1500px; margin: 0 auto; padding: 24px 20px 80px; }}
h1 {{ font-size: 1.5rem; margin: 0 0 4px; }}
h2 {{ font-size: 1.15rem; margin: 36px 0 12px; border-bottom: 1px solid var(--border); padding-bottom: 6px; }}
h3 {{ font-size: 1rem; margin: 0 0 8px; color: var(--accent); }}
.subtitle {{ color: var(--muted); margin-bottom: 20px; }}
.badge {{ display: inline-block; background: var(--accent-soft); color: var(--accent); padding: 2px 10px; border-radius: 999px; font-size: 0.8rem; font-weight: 600; margin-right: 6px; }}
table {{ border-collapse: collapse; width: 100%; background: var(--panel); }}
.table-scroll {{ overflow-x: auto; border: 1px solid var(--border); border-radius: 8px; }}
th, td {{ border: 1px solid var(--border); padding: 6px 9px; text-align: left; font-size: 0.82rem; vertical-align: top; }}
th {{ background: var(--accent-soft); color: var(--text); position: sticky; top: 0; z-index: 2; }}
.metrics-table th, .metrics-table td {{ text-align: center; }}
.metrics-table td:first-child, .metrics-table th:first-child {{ text-align: left; }}
.muted {{ color: var(--muted); }}
.small {{ font-size: 0.76rem; }}
.note {{ color: var(--muted); font-size: 0.82rem; margin: 6px 0 14px; }}
.two-col {{ display: flex; gap: 24px; flex-wrap: wrap; }}
.two-col > div {{ flex: 1; min-width: 300px; }}
.dec-badge {{ display: inline-block; padding: 2px 9px; border-radius: 6px; font-size: 0.76rem; font-weight: 700; color: #fff; }}
.dec-hot {{ background: var(--hot); }}
.dec-warm {{ background: var(--warm); }}
.dec-watch {{ background: var(--watch); color: #2b2410; }}
.dec-none {{ background: var(--none); }}
.dec-competitor {{ background: var(--competitor); }}
.ok-badge {{ padding: 2px 8px; border-radius: 6px; font-size: 0.76rem; font-weight: 700; }}
.ok-yes {{ background: color-mix(in srgb, var(--ok) 18%, transparent); color: var(--ok); }}
.ok-no {{ background: color-mix(in srgb, var(--bad) 18%, transparent); color: var(--bad); }}
tr.row-bad {{ background: var(--row-bad-bg); }}
.author-line {{ font-style: italic; color: var(--muted); margin-bottom: 4px; }}
.content-text {{ white-space: pre-wrap; max-width: 360px; }}
.col-id {{ font-weight: 700; white-space: nowrap; }}
.filters {{ display: flex; gap: 10px; flex-wrap: wrap; align-items: center; margin: 14px 0; background: var(--panel); border: 1px solid var(--border); border-radius: 8px; padding: 12px; }}
.filters label {{ font-size: 0.82rem; color: var(--muted); margin-right: 4px; }}
.filters select, .filters input[type=text] {{ background: var(--bg); color: var(--text); border: 1px solid var(--border); border-radius: 6px; padding: 5px 8px; font-size: 0.82rem; }}
.filters input[type=text] {{ min-width: 220px; }}
#rowCount {{ font-weight: 700; color: var(--accent); }}
.profile-card {{ background: var(--panel); border: 1px solid var(--border); border-radius: 8px; padding: 14px 18px; margin-bottom: 14px; }}
.profile-card ul {{ margin: 4px 0 10px 18px; padding: 0; }}
.profile-card p {{ margin: 4px 0; font-size: 0.88rem; }}
pre.rule-block {{ background: var(--panel); border: 1px solid var(--border); border-radius: 8px; padding: 16px; overflow-x: auto; white-space: pre-wrap; font-size: 0.82rem; line-height: 1.5; }}
.conf-diag {{ background: color-mix(in srgb, var(--ok) 20%, transparent); font-weight: 700; }}
.conf-off {{ background: color-mix(in srgb, var(--bad) 16%, transparent); }}
footer {{ margin-top: 40px; color: var(--muted); font-size: 0.78rem; text-align: center; }}
</style>
</head>
<body>
<div class="wrap">
  <h1>Lead qualification rule &mdash; regression test report</h1>
  <div class="subtitle">
    <span class="badge">{total_rows} judgments</span>
    <span class="badge">{'ALL TARGETS MET' if all_target_pass else 'BELOW TARGET'}</span>
    {'<span class="badge">' + ('AT OR ABOVE BASELINE' if all_baseline_pass else 'REGRESSION VS BASELINE') + '</span>' if all_baseline_pass is not None else ''}
    &nbsp;&middot;&nbsp; Lead acc: <b>{score.fmt_pct(metrics['lead_acc'])}</b>
    &nbsp;&middot;&nbsp; Tier acc: <b>{score.fmt_pct(metrics['tier_acc'])}</b>
    &nbsp;&middot;&nbsp; Group A warm: <b>{score.fmt_pct(metrics['groupA_warm'])}</b>
    &nbsp;&middot;&nbsp; Group E competitor: <b>{score.fmt_pct(metrics['groupE_competitor'])}</b>
    &nbsp;&middot;&nbsp; Group F/G none: <b>{score.fmt_pct(metrics['groupFG_none'])}</b>
  </div>
  {missing_note}
  {warnings_note}

  <h2>Targets</h2>
  <div class="table-scroll">
  <table class="metrics-table">
    <thead><tr><th>Metric</th><th>Value</th><th>Target</th><th>Result</th></tr></thead>
    <tbody>{target_rows()}</tbody>
  </table>
  </div>
  {baseline_section}

  <h2>Accuracy by client</h2>
  <div class="two-col">
    <div>
      <table class="metrics-table"><thead><tr><th>Client</th><th>Lead ok</th><th>Lead %</th><th>Tier ok</th><th>Tier %</th></tr></thead><tbody>{client_acc_rows}</tbody></table>
    </div>
    <div>
      <h3>By group</h3>
      <table class="metrics-table"><thead><tr><th>Group</th><th>Lead ok</th><th>Lead %</th><th>Tier ok</th><th>Tier %</th></tr></thead><tbody>{group_acc_rows}</tbody></table>
    </div>
  </div>

  <h2>Confusion matrix: expected &rarr; judged</h2>
  <div class="table-scroll">
  <table class="metrics-table">
    <thead><tr><th>Expected \\ Judged</th>{conf_header}</tr></thead>
    <tbody>{confusion_table}</tbody>
  </table>
  </div>

  <h2>Filters</h2>
  <div class="filters">
    <label>Client: <select id="fClient">
      <option value="">All</option>
      {"".join(f'<option value="{c}">{esc(client_names[c])}</option>' for c in clients)}
    </select></label>
    <label>Group: <select id="fGroup">
      <option value="">All</option>
      {"".join(f'<option value="{g}">{esc(g)}</option>' for g in sorted(metrics['byGroup'].keys()))}
    </select></label>
    <label><input type="checkbox" id="fWrongOnly"> Wrong only</label>
    <label>Search: <input type="text" id="fText" placeholder="keyword..."></label>
    <span>Showing: <span id="rowCount">{total_rows}</span> / {total_rows} rows</span>
  </div>

  <h2>Row detail (scenario &times; client)</h2>
  <div class="table-scroll">
  <table id="mainTable">
    <thead>
      <tr>
        <th>ID</th><th>Group</th><th>Industry / Person</th><th>Surface / Community</th><th>Content</th>
        <th>Client</th><th>Expected</th><th>Judged</th><th>Judge's reasons</th><th>OK</th>
      </tr>
    </thead>
    <tbody id="mainBody">{main_rows_html}</tbody>
  </table>
  </div>

  <h2>Client buyer profiles</h2>
  {profiles_html}

  <h2>Rule text scored</h2>
  <pre class="rule-block">{esc(rule_text)}</pre>

  <footer>Solo Agency &mdash; lead rule regression report &middot; {total_rows} rows</footer>
</div>

<script>
(function() {{
  var fClient = document.getElementById('fClient');
  var fGroup = document.getElementById('fGroup');
  var fWrongOnly = document.getElementById('fWrongOnly');
  var fText = document.getElementById('fText');
  var rowCount = document.getElementById('rowCount');
  var rows = Array.prototype.slice.call(document.querySelectorAll('#mainBody tr.data-row'));

  function apply() {{
    var client = fClient.value;
    var group = fGroup.value;
    var wrongOnly = fWrongOnly.checked;
    var text = fText.value.trim().toLowerCase();
    var count = 0;
    rows.forEach(function(r) {{
      var show = true;
      if (client && r.getAttribute('data-client') !== client) show = false;
      if (group && r.getAttribute('data-group') !== group) show = false;
      if (wrongOnly && r.getAttribute('data-ok') === '1') show = false;
      if (text && r.getAttribute('data-search').indexOf(text) === -1) show = false;
      r.hidden = !show;
      if (show) count++;
    }});
    rowCount.textContent = count;
  }}
  fClient.addEventListener('change', apply);
  fGroup.addEventListener('change', apply);
  fWrongOnly.addEventListener('change', apply);
  fText.addEventListener('input', apply);
}})();
</script>
</body>
</html>
"""

    os.makedirs(os.path.dirname(os.path.abspath(args.out)) or ".", exist_ok=True)
    with open(args.out, "w", encoding="utf-8") as f:
        f.write(html_out)

    print("OK")
    print("out", args.out)
    print("size_bytes", os.path.getsize(args.out))
    print("total_rows", total_rows)
    print("lead_acc", metrics["lead_acc"])
    print("tier_acc", metrics["tier_acc"])


if __name__ == "__main__":
    main()
