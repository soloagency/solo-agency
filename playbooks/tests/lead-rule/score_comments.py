#!/usr/bin/env python3
"""Deterministic scorer for the batch comment-triage test (Decision 3,
`playbooks/COMMENT_TRIAGE_RULE.md`).

usage: score_comments.py <comment_threads.json> <judgments_dir> [--show N]

All paths may be relative to the current working directory or absolute —
run it from this directory (`playbooks/tests/lead-rule/`) as:

    python3 score_comments.py comment_threads.json baseline/comment_triage

`<comment_threads.json>` is this directory's `comment_threads.json` (24
threads x ~40 authors each, two labeled clients per thread: `client` and
`secondary_client`). `<judgments_dir>` holds judgment files named
`{post}_{client}_{from}.json` (e.g. `P01_realtor_0.json`), each a JSON
array or `{"results": [...]}` of `playbooks/COMMENT_TRIAGE_RULE.md`'s
Output objects keyed by row `id`.
"""
import json, sys, glob, os, collections

def main():
    th = json.load(open(sys.argv[1], encoding='utf-8'))
    jd = sys.argv[2]
    show = int(sys.argv[4]) if len(sys.argv) > 4 and sys.argv[3] == '--show' else 0
    J = {}
    for f in glob.glob(os.path.join(jd, '*.json')):
        name = os.path.basename(f)[:-5]
        client = next((k for k in ['solo_agency','ai_video','realtor','insurance','mlo'] if ('_' + k + '_') in name), None)
        try:
            d = json.load(open(f, encoding='utf-8'))
        except Exception as e:
            print('bad file', f, e); continue
        rs = d.get('results', d) if isinstance(d, dict) else d
        for r in rs:
            if not isinstance(r, dict) or 'id' not in r or 'decision' not in r: continue
            J.setdefault(r['id'], {})[client] = r
    m = dict(total=0, ok=0, keep_exp=0, keep_tp=0, keep_pred=0, comp_exp=0, comp_drop=0, noise_exp=0, noise_drop=0, missing=0)
    by = collections.defaultdict(lambda: dict(total=0, ok=0)); fails = []; conf = collections.Counter()
    for t in th:
        for r in t['rows']:
            for which, c in (('primary', t['client']), ('secondary', t.get('secondary_client'))):
                e = r['expected'].get(which)
                if not e or not c: continue
                j = J.get(r['id'], {}).get(c)
                m['total'] += 1; by[c]['total'] += 1
                if not j: m['missing'] += 1; continue
                if e['decision'] == 'keep':
                    m['keep_exp'] += 1
                    if j['decision'] == 'keep': m['keep_tp'] += 1
                if j['decision'] == 'keep': m['keep_pred'] += 1
                if e.get('competitor'):
                    m['comp_exp'] += 1
                    if j['decision'] == 'drop': m['comp_drop'] += 1
                if e.get('noise'):
                    m['noise_exp'] += 1
                    if j['decision'] == 'drop': m['noise_drop'] += 1
                if e['decision'] == j['decision']: m['ok'] += 1; by[c]['ok'] += 1
                else:
                    conf[(e['decision'], j['decision'], 'fit=' + j.get('fit', '?'))] += 1
                    fails.append((r['id'], c, e['decision'], j['decision'], j.get('fit'), r['author_line'][:40], ' | '.join(r['comments'])[:110], j.get('reason', '')[:120]))
    acc = m['ok'] / m['total']; rec = m['keep_tp'] / m['keep_exp'] if m['keep_exp'] else 1; prec = m['keep_tp'] / m['keep_pred'] if m['keep_pred'] else 1
    cd = m['comp_drop'] / m['comp_exp'] if m['comp_exp'] else 1; nd = m['noise_drop'] / m['noise_exp'] if m['noise_exp'] else 1
    print(f"total {m['total']} missing {m['missing']} | acc {acc*100:.1f}% keep-recall {rec*100:.1f}% keep-precision {prec*100:.1f}% competitor-drop {cd*100:.1f}% noise-drop {nd*100:.1f}%")
    print('by client:', {k: f"{v['ok']}/{v['total']}" for k, v in by.items()})
    print('confusions:', dict(conf))
    passed = acc >= 0.93 and rec >= 0.95 and prec >= 0.90 and cd >= 0.95 and nd >= 0.95
    print('PASS' if passed else 'FAIL (targets: acc>=93, keep-recall>=95, keep-precision>=90, comp>=95, noise>=95)')
    for f in fails[:show]:
        print('  ', f)
    return 0 if passed else 1

if __name__ == '__main__':
    sys.exit(main())
