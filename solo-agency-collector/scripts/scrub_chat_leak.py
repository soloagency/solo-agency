#!/usr/bin/env python3
"""Remove Messenger chat text that the collector captured into harvest and CRM files.

Until 2026-09-18 the page extractor read the whole document body, so the Facebook Messenger
drawer — an open conversation, its read receipts and any link shared in it — was captured as
if it were a post of the page being scanned. filtering.js now scopes the read to the page's
content landmark, refuses off-platform links and skips chat surfaces; this script cleans what
the old code already wrote.

It edits three shapes, and only these:
  * a "Post N" block whose Content is a chat transcript  -> the whole block is dropped
  * a stray chat line ("Seen by X at ...", "Message sent ... by X", "Enter, Message sent")
  * a candidate url that is a /messages/ thread, or an off-platform url promoted to
    profile_url/post_url by a dropped block

Real posts are never touched: a block is dropped only when its own content matches a chat
marker. Every changed file is copied to the backup directory first, and every changed record
gains a `chat_scrub` note saying what went and when, so the edit is visible and reversible.

Usage:
  scrub_chat_leak.py --root <daily-content-pipeline dir> [--apply] [--backup-dir DIR]
Dry run by default: it reports what it WOULD change and writes nothing.
"""
import argparse, json, os, re, shutil, sys, datetime

SEEN = re.compile(r"Seen by [^\n]{2,60}? at ")
BODY = re.compile(r"At [0-9][^,\n]{2,28}, [^:\n]{1,28}: ")
SENT = re.compile(r"Message sent [^\n]{0,40} by ")
UI   = re.compile(r"Enter, Message sent|Open Attachment,")
CHAT_URL = re.compile(r"/messages/(e2ee/)?t/")
POST_SPLIT = re.compile(r"(?m)^(?=Post \d+\s*$)")

def is_chat_line(line):
    return bool(SEEN.search(line) or BODY.search(line) or SENT.search(line) or UI.search(line))

def chat_score(text):
    return sum(1 for ln in text.splitlines() if is_chat_line(ln))

def scrub_text(text):
    """Drop chat post-blocks and stray chat lines. Returns (new_text, blocks, lines)."""
    if not text or not isinstance(text, str):
        return text, 0, 0
    blocks = POST_SPLIT.split(text)
    kept, dropped_blocks = [], 0
    for b in blocks:
        # A block is chat when its own body carries the markers, not because a sibling did.
        if b.strip().startswith("Post ") and chat_score(b) >= 2:
            dropped_blocks += 1
            continue
        kept.append(b)
    out = "".join(kept)
    lines, dropped_lines = [], 0
    for ln in out.splitlines():
        if is_chat_line(ln):
            dropped_lines += 1
            continue
        lines.append(ln)
    out = "\n".join(lines)
    out = re.sub(r"\n{3,}", "\n\n", out).strip()
    return out, dropped_blocks, dropped_lines

def scrub_urls(urls, page_host_ok):
    keep, removed = [], []
    for u in urls or []:
        s = str(u or "")
        if CHAT_URL.search(s) or (s.startswith("http") and not page_host_ok(s)):
            removed.append(s)
        else:
            keep.append(u)
    return keep, removed

def host_checker(source_url):
    """Only reject an off-platform url when the page itself belongs to a platform."""
    GROUPS = {"facebook": ("facebook.com", "fb.watch", "fb.com"), "x": ("x.com", "twitter.com"),
              "instagram": ("instagram.com",), "linkedin": ("linkedin.com",), "zillow": ("zillow.com",)}
    def group_of(u):
        try:
            h = re.sub(r"^www\.", "", (re.split(r"/+", str(u).split("://", 1)[-1])[0] or "").lower())
        except Exception:
            return ""
        for k, doms in GROUPS.items():
            if any(h == d or h.endswith("." + d) for d in doms):
                return k
        return ""
    page = group_of(source_url)
    if not page:
        return lambda u: True          # a page belonging to no platform keeps every link
    return lambda u: group_of(u) in ("", page)

def scrub_record(rec, wide_urls=False):
    """Returns (changed, note) and mutates rec in place.

    URLs are touched only where the leak actually reached: a /messages/ thread url always, and
    an off-platform url only in a record that DID carry chat (its post block was just dropped,
    which is how such a url was promoted to profile_url in the first place). Without that rule
    the pass would also rewrite thousands of records that never had a chat problem — a data
    rewrite nobody asked for. --wide-urls opts into the broader cleanup deliberately.
    """
    note = {"blocks": 0, "lines": 0, "urls": []}
    ok = host_checker(rec.get("source_url") or rec.get("current_url") or "")
    for field in ("raw_visible_text_excerpt", "visible_text_summary", "text", "meaningful_text"):
        if isinstance(rec.get(field), str):
            new, b, l = scrub_text(rec[field])
            if new != rec[field]:
                rec[field] = new
                note["blocks"] += b
                note["lines"] += l
    if isinstance(rec.get("raw_visible_text_excerpt"), str) and isinstance(rec.get("visible_text_summary"), str):
        rec["visible_text_summary"] = rec["raw_visible_text_excerpt"][:1200]
    had_chat = bool(note["blocks"] or note["lines"])
    if not (had_chat or wide_urls):
        ok = lambda u: True          # no chat here: leave every url exactly as it was
    for field in ("post_candidates", "profile_candidates", "entity_candidates"):
        if isinstance(rec.get(field), list):
            if field == "entity_candidates":
                before = len(rec[field])
                rec[field] = [it for it in rec[field]
                              if not (isinstance(it, dict) and (CHAT_URL.search(str(it.get("url", ""))) or not ok(str(it.get("url", "")))))]
                if len(rec[field]) != before:
                    note["urls"].append("%s:-%d" % (field, before - len(rec[field])))
            else:
                keep, removed = scrub_urls(rec[field], ok)
                if removed:
                    rec[field] = keep
                    note["urls"] += removed
    for field in ("profile_url", "post_url"):
        v = str(rec.get(field) or "")
        if v and (CHAT_URL.search(v) or not ok(v)):
            note["urls"].append("%s=%s" % (field, v))
            rec[field] = rec.get("source_url") or ""
    changed = bool(note["blocks"] or note["lines"] or note["urls"])
    if changed:
        rec["chat_scrub"] = {"at": datetime.datetime.now().astimezone().isoformat(),
                             "reason": "messenger drawer captured by the pre-2026-09-18 whole-body extractor",
                             "post_blocks_removed": note["blocks"], "lines_removed": note["lines"],
                             "urls_removed": note["urls"][:20]}
    return changed, note

def walk_json(obj, fn):
    if isinstance(obj, dict):
        return {k: walk_json(v, fn) for k, v in obj.items()}
    if isinstance(obj, list):
        return [walk_json(v, fn) for v in obj]
    if isinstance(obj, str):
        return fn(obj)
    return obj

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", required=True, help="daily-content-pipeline directory")
    ap.add_argument("--apply", action="store_true", help="write the changes (default: dry run)")
    ap.add_argument("--backup-dir", default="")
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--include-backups", action="store_true",
                    help="also rewrite .backup/, backups/, *_before/ and dated snapshot dirs "
                         "(default: left alone — a backup exists to be restorable)")
    ap.add_argument("--wide-urls", action="store_true",
                    help="also drop off-platform urls from records that carried no chat")
    a = ap.parse_args()
    stamp = datetime.datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
    backup = a.backup_dir or os.path.join(a.root, "..", "backups", "chat_scrub_" + stamp)
    totals = {"files": 0, "records": 0, "blocks": 0, "lines": 0, "urls": 0, "strings": 0}
    changed_files = []

    def backup_file(p):
        if not a.apply:
            return
        dest = os.path.join(backup, os.path.relpath(p, a.root))
        os.makedirs(os.path.dirname(dest), exist_ok=True)
        if not os.path.exists(dest):
            shutil.copy2(p, dest)

    BACKUPISH = re.compile(r"\.backup/|/backups/|_before/|contaminated|pending_backups|_20\d{6}_|_20\d{2}-\d{2}-\d{2}_")
    for dirpath, _, files in os.walk(a.root):
        if not a.include_backups and BACKUPISH.search(dirpath + "/"):
            continue
        for fn in files:
            p = os.path.join(dirpath, fn)
            if a.limit and totals["files"] >= a.limit:
                break
            try:
                if fn.endswith(".jsonl"):
                    lines_out, hits, blocks, lns, urls = [], 0, 0, 0, 0
                    dirty = False
                    for line in open(p, encoding="utf-8", errors="replace"):
                        try:
                            rec = json.loads(line)
                        except Exception:
                            lines_out.append(line.rstrip("\n")); continue
                        ch, note = scrub_record(rec, a.wide_urls)
                        if ch:
                            dirty = True; hits += 1; blocks += note["blocks"]; lns += note["lines"]; urls += len(note["urls"])
                        lines_out.append(json.dumps(rec, ensure_ascii=False))
                    if dirty:
                        changed_files.append((p, hits, blocks, lns, urls))
                        totals["files"] += 1; totals["records"] += hits
                        totals["blocks"] += blocks; totals["lines"] += lns; totals["urls"] += urls
                        if a.apply:
                            backup_file(p)
                            open(p, "w", encoding="utf-8").write("\n".join(lines_out) + "\n")
                elif fn.endswith(".json"):
                    raw = open(p, encoding="utf-8", errors="replace").read()
                    if not (SEEN.search(raw) or BODY.search(raw) or UI.search(raw)):
                        continue
                    d = json.loads(raw)
                    n = [0]
                    def fix(s):
                        new, b, l = scrub_text(s)
                        if new != s:
                            n[0] += b + l
                        return new
                    d2 = walk_json(d, fix)
                    if n[0]:
                        changed_files.append((p, 1, 0, n[0], 0))
                        totals["files"] += 1; totals["strings"] += n[0]
                        if a.apply:
                            backup_file(p)
                            json.dump(d2, open(p, "w", encoding="utf-8"), ensure_ascii=False, indent=2)
                elif fn.endswith(".html"):
                    raw = open(p, encoding="utf-8", errors="replace").read()
                    if not (SEEN.search(raw) or BODY.search(raw) or UI.search(raw)):
                        continue
                    new = "\n".join(ln for ln in raw.splitlines() if not is_chat_line(ln))
                    new = re.sub(r'<li>[^<]*(?:Profile|Post/current) candidate: <a href="[^"]*(?:x\.com|/messages/)[^"]*"[^>]*>[^<]*</a></li>', "", new)
                    if new != raw:
                        removed = len(raw.splitlines()) - len(new.splitlines())
                        changed_files.append((p, 1, 0, removed, 0))
                        totals["files"] += 1; totals["lines"] += removed
                        if a.apply:
                            backup_file(p)
                            open(p, "w", encoding="utf-8").write(new)
            except Exception as e:
                print("  skip %s (%s)" % (p, e), file=sys.stderr)

    mode = "APPLIED" if a.apply else "DRY RUN (nothing written)"
    print("== chat leak scrub — %s ==" % mode)
    print("files changed   : %d" % totals["files"])
    print("records changed : %d" % totals["records"])
    print("post blocks gone: %d" % totals["blocks"])
    print("lines gone      : %d  (+%d inside json strings)" % (totals["lines"], totals["strings"]))
    print("urls removed    : %d" % totals["urls"])
    if a.apply:
        print("backup          : %s" % os.path.normpath(backup))
    print("\ntop files:")
    for p, h, b, l, u in sorted(changed_files, key=lambda r: -(r[2] * 10 + r[3]))[:25]:
        print("  %-96s rec=%d blocks=%d lines=%d urls=%d" % (os.path.relpath(p, a.root)[:96], h, b, l, u))
    print("\n(%d files total)" % len(changed_files))

if __name__ == "__main__":
    main()
