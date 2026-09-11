#!/usr/bin/env python3
"""Deterministic scorer for the Step-5 comment_source test
(`playbooks/LEAD_QUALIFICATION_RULE.md` Step 5).

usage: score_comment_source.py <comment_posts.json> <comment_posts_expected.json> <judgments_dir> [--show N]

All paths may be relative to the current working directory or absolute —
run it from this directory (`playbooks/tests/lead-rule/`) as:

    python3 score_comment_source.py comment_posts.json comment_posts_expected.json baseline/comment_source

`<comment_posts.json>` is this directory's `comment_posts.json` (69 posts).
`<comment_posts_expected.json>` is `comment_posts_expected.json`, keyed by
post id -> {client_key: "likely"|"unlikely"}. `<judgments_dir>` holds
judgment files named `{client}_{chunk}.json` with `results[]` of
`{id, comment_source, comment_source_reason}` objects (the rule's Step 5
Output fields).
"""
import json, sys, glob, os, collections

def main():
    posts = {p['id']: p for p in json.load(open(sys.argv[1], encoding='utf-8'))}
    exp = json.load(open(sys.argv[2], encoding='utf-8'))
    jd = sys.argv[3]
    show = int(sys.argv[5]) if len(sys.argv) > 5 and sys.argv[4] == '--show' else 0
    J = {}
    for f in glob.glob(os.path.join(jd, '*.json')):
        client = next((k for k in ['solo_agency','ai_video','realtor','insurance','mlo'] if os.path.basename(f).startswith(k + '_')), None)
        d = json.load(open(f, encoding='utf-8')); rs = d.get('results', d) if isinstance(d, dict) else d
        for r in rs: J.setdefault(r['id'], {})[client] = r
    total = ok = le = tp = lp = 0; by = collections.defaultdict(lambda: dict(total=0, ok=0)); fails = []
    for pid, cs in exp.items():
        for c, e in cs.items():
            j = J.get(pid, {}).get(c); total += 1; by[c]['total'] += 1
            if not j: continue
            if e == 'likely':
                le += 1
                if j['comment_source'] == 'likely': tp += 1
            if j['comment_source'] == 'likely': lp += 1
            if e == j['comment_source']: ok += 1; by[c]['ok'] += 1
            else: fails.append((pid, c, e, j['comment_source'], posts.get(pid, {}).get('text', '')[:80], j.get('comment_source_reason', '')[:110]))
    acc = ok / total; rec = tp / le if le else 1; prec = tp / lp if lp else 1
    print(f"total {total} | acc {acc*100:.1f}% likely-recall {rec*100:.1f}% likely-precision {prec*100:.1f}% (expected likely {le})")
    print('by client:', {k: f"{v['ok']}/{v['total']}" for k, v in by.items()})
    passed = acc >= 0.93 and rec >= 0.90 and prec >= 0.90
    print('PASS' if passed else 'FAIL (targets: acc>=93, likely-recall>=90, likely-precision>=90)')
    for f in fails[:show]: print('  ', f)
    return 0 if passed else 1

if __name__ == '__main__':
    sys.exit(main())
