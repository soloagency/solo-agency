#!/usr/bin/env python3
"""Split dataset.json into blind chunks for a fresh regression-test run.

Strips group/trap/expected/label_notes from every scenario, assigns a
sequential blind id (s001, s002, ...) in dataset order, and writes:

  <out-dir>/chunk_01.json, chunk_02.json, ...   (lists of blind scenarios)
  <out-dir>/map.json                            (blind id -> real dataset id)

Each blind scenario keeps only: id, platform, surface, community,
author_line, text, date_note — exactly what the judge should see, nothing
that reveals the group, the trap, or the expected answer.

Usage:
    python3 split_blind.py --dataset dataset.json \
        --out-dir runs/2026-09-11/blind --chunk-size 10
"""

import argparse
import json
import os

BLIND_FIELDS = ["id", "platform", "surface", "community", "author_line",
                "text", "date_note"]


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                  formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--dataset", required=True, help="path to dataset.json")
    ap.add_argument("--out-dir", required=True,
                     help="directory to write chunk_NN.json and map.json into")
    ap.add_argument("--chunk-size", type=int, default=10)
    args = ap.parse_args()

    with open(args.dataset, "r", encoding="utf-8") as f:
        dataset = json.load(f)

    os.makedirs(args.out_dir, exist_ok=True)

    blind_map = {}
    chunks = []
    current = []
    for i, scen in enumerate(dataset):
        blind_id = f"s{i + 1:03d}"
        blind_map[blind_id] = scen["id"]
        row = {"id": blind_id}
        for field in BLIND_FIELDS[1:]:
            row[field] = scen.get(field, "")
        current.append(row)
        if len(current) == args.chunk_size:
            chunks.append(current)
            current = []
    if current:
        chunks.append(current)

    for idx, chunk in enumerate(chunks, start=1):
        fp = os.path.join(args.out_dir, f"chunk_{idx:02d}.json")
        with open(fp, "w", encoding="utf-8") as f:
            json.dump(chunk, f, indent=2, ensure_ascii=False)

    map_fp = os.path.join(args.out_dir, "map.json")
    with open(map_fp, "w", encoding="utf-8") as f:
        json.dump(blind_map, f, indent=2, ensure_ascii=False)

    print(f"wrote {len(chunks)} chunks ({args.chunk_size} scenarios each, "
          f"last chunk may be shorter) and map.json to {args.out_dir}")
    print(f"total scenarios: {len(dataset)}")


if __name__ == "__main__":
    main()
