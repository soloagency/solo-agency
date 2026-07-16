#!/usr/bin/env python3
"""Reusable OutreachCRM report renderer.

The tool intentionally uses Python's standard library so scheduled agents can
render reports without installing dependencies. It has two commands:

  render  - Markdown/source record to polished standalone HTML, optional PDF.
  package - Combine scrubbed staging HTML files into the single client-facing HTML/PDF.
"""

from __future__ import annotations

import argparse
import datetime as dt
import html
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import urllib.parse
from html.parser import HTMLParser
from pathlib import Path
from typing import Iterable


# Terms that must never appear in a client-facing deliverable (only the weekly
# client report is client-facing). Keep this list in sync with docs/DESIGN.md §19.
CLIENT_BLIND_TERMS = [
    "OutreachCRM",
    "WideCast",
    "INTERNAL_REPORT",
    "MCP",
    "OpenAPI",
    "API key",
    "Telegram",
    "PDNA",
    "provider_config",
    "Client tools",
    "global MCP",
    "automation",
    "scheduled task",
    "agent debug",
    "config file",
    "debug",
    # OutreachCRM internal vocabulary
    "sendbox",
    "gmail_client",
    "crm_store",
    "storage_config",
    "trk.",
    "HMAC",
    "token.json",
    "sent_log",
    "suppression",
    "warmup",
    "quota",
    "guessed",
]


CSS = r"""
:root {
  --paper: #f7f4ef;
  --ink: #171512;
  --muted: #6d665c;
  --line: #ddd5ca;
  --panel: #fffdf8;
  --panel-strong: #ece4d8;
  --accent: #0f766e;
  --accent-ink: #083b36;
  --accent-soft: #d9efea;
  --danger: #a3412f;
  --shadow: 0 24px 70px rgba(38, 32, 24, .12);
  color-scheme: light;
}

* { box-sizing: border-box; }
html { min-width: 0; background: var(--paper); }
body {
  margin: 0;
  min-width: 0;
  color: var(--ink);
  background:
    linear-gradient(135deg, rgba(15, 118, 110, .09), transparent 28rem),
    radial-gradient(circle at 88% 8%, rgba(163, 65, 47, .11), transparent 20rem),
    var(--paper);
  font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", Arial, sans-serif;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

a { color: var(--accent-ink); text-underline-offset: .16em; }
code {
  padding: .14rem .34rem;
  border-radius: .35rem;
  background: rgba(23, 21, 18, .07);
  font-size: .92em;
}
pre {
  overflow-x: auto;
  padding: 1rem;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: #211d18;
  color: #fff8ed;
}
pre code { padding: 0; background: transparent; color: inherit; }

.report-page {
  width: min(1180px, calc(100% - 36px));
  margin: 0 auto;
  padding: 28px 0 64px;
}

.report-hero {
  min-height: min(82dvh, 760px);
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(280px, .65fr);
  gap: clamp(24px, 5vw, 72px);
  align-items: center;
  padding: clamp(34px, 7vw, 88px) 0 34px;
}

.eyebrow {
  margin: 0 0 1rem;
  color: var(--accent-ink);
  font-size: .78rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: .08em;
}

h1 {
  max-width: 11ch;
  margin: 0;
  font-size: clamp(3rem, 9vw, 7.4rem);
  line-height: .92;
  letter-spacing: 0;
}

.dek {
  max-width: 66ch;
  margin: 1.35rem 0 0;
  color: #38332d;
  font-size: clamp(1.08rem, 2.2vw, 1.45rem);
}

.hero-panel {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(23, 21, 18, .12);
  border-radius: 8px;
  background: rgba(255, 253, 248, .82);
  box-shadow: var(--shadow);
}
.hero-panel::before {
  content: "";
  display: block;
  height: 12px;
  background: linear-gradient(90deg, var(--accent), #d97706, #a3412f);
}
.hero-panel-inner { padding: 22px; }
.meta-grid {
  display: grid;
  gap: 14px;
}
.meta-item {
  padding: 14px 0;
  border-bottom: 1px solid var(--line);
}
.meta-item:last-child { border-bottom: 0; }
.meta-label {
  display: block;
  color: var(--muted);
  font-size: .76rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: .08em;
}
.meta-value {
  display: block;
  margin-top: .22rem;
  font-size: 1.05rem;
  font-weight: 750;
}

.report-shell {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 42px;
  align-items: start;
}
.toc {
  position: sticky;
  top: 18px;
  padding: 16px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: rgba(255, 253, 248, .72);
}
.toc-title {
  margin: 0 0 .8rem;
  color: var(--muted);
  font-size: .72rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: .08em;
}
.toc a {
  display: block;
  padding: .46rem 0;
  color: var(--ink);
  font-size: .92rem;
  font-weight: 650;
  text-decoration: none;
  border-top: 1px solid rgba(221, 213, 202, .7);
}
.toc a:first-of-type { border-top: 0; }

.report-body {
  min-width: 0;
}
.report-section {
  margin: 0 0 34px;
  padding: clamp(22px, 4vw, 40px);
  border: 1px solid var(--line);
  border-radius: 8px;
  background: rgba(255, 253, 248, .86);
  box-shadow: 0 14px 42px rgba(38, 32, 24, .07);
}
.report-section h2 {
  max-width: 17ch;
  margin: 0 0 1rem;
  font-size: clamp(1.9rem, 4.6vw, 4rem);
  line-height: .98;
  letter-spacing: 0;
}
.report-section h3 {
  margin: 2rem 0 .7rem;
  font-size: clamp(1.18rem, 2vw, 1.55rem);
  line-height: 1.12;
}
.report-section h4 {
  margin: 1.6rem 0 .55rem;
  font-size: 1.02rem;
}
.report-section p { margin: .78rem 0; }
.report-section > p:first-child { margin-top: 0; }

.lead-paragraph {
  color: #38332d;
  font-size: 1.12rem;
}

ul, ol { padding-left: 1.2rem; }
li { margin: .4rem 0; }
blockquote {
  margin: 1.35rem 0;
  padding: .75rem 0 .75rem 1rem;
  border-left: 4px solid var(--accent);
  color: #37322c;
}

.table-scroll {
  width: 100%;
  overflow-x: auto;
  margin: 1.2rem 0;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--panel);
}
table {
  width: 100%;
  min-width: 680px;
  border-collapse: collapse;
  font-size: .94rem;
}
th, td {
  padding: .82rem .9rem;
  text-align: left;
  vertical-align: top;
  border-bottom: 1px solid var(--line);
}
th {
  color: var(--accent-ink);
  background: var(--accent-soft);
  font-size: .76rem;
  font-weight: 850;
  text-transform: uppercase;
  letter-spacing: .06em;
}
tr:last-child td { border-bottom: 0; }

.status-note {
  margin: 28px 0 0;
  padding: 14px 16px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: rgba(217, 239, 234, .7);
  color: var(--accent-ink);
  font-weight: 650;
}

.print-source-title {
  margin: 0 0 24px;
  padding: 18px 20px;
  border-left: 6px solid var(--accent);
  background: var(--panel);
}

.draft-actions { margin: .4rem 0 .9rem; }
.copy-btn {
  padding: .5rem .95rem;
  border: 1px solid var(--accent-ink);
  border-radius: 6px;
  background: var(--accent-soft);
  color: var(--accent-ink);
  font-size: .88rem;
  font-weight: 700;
  cursor: pointer;
}
.copy-btn:hover { background: var(--accent); color: #fff; }
.copy-btn.copied { background: var(--accent); color: #fff; }
.draft-editable {
  padding: 14px 16px;
  border: 1px dashed var(--accent-ink);
  border-radius: 8px;
  background: var(--panel);
}
.draft-editable:focus { outline: 2px solid var(--accent); outline-offset: 2px; }

@media (max-width: 860px) {
  .report-page { width: min(100% - 28px, 720px); padding-top: 12px; }
  .report-hero { min-height: auto; grid-template-columns: 1fr; padding: 34px 0 20px; }
  h1 { max-width: 10.5ch; font-size: clamp(3rem, 17vw, 5.6rem); }
  .report-shell { grid-template-columns: 1fr; gap: 22px; }
  .toc { position: static; }
  .toc a { display: inline-block; margin-right: 14px; border-top: 0; }
  .report-section { padding: 22px; }
}

@media print {
  @page { margin: 16mm 14mm; }
  html, body { background: #ffffff !important; color: #111111; }
  body { font-size: 10.5pt; }
  .report-page { width: 100%; padding: 0; }
  .report-hero { min-height: 0; display: block; padding: 0 0 18mm; }
  h1 { max-width: none; font-size: 36pt; line-height: 1; }
  .dek { font-size: 13pt; }
  .hero-panel, .report-section, .toc, .table-scroll {
    box-shadow: none !important;
    background: #ffffff !important;
  }
  .toc { display: none; }
  .report-shell { display: block; }
  .report-section {
    page-break-inside: avoid;
    break-inside: avoid;
    margin: 0 0 10mm;
    padding: 8mm 0;
    border-width: 1px 0 0;
    border-radius: 0;
  }
  .report-section h2 { font-size: 24pt; max-width: none; }
  .table-scroll { overflow: visible; border: 0; }
  table { min-width: 0; font-size: 8.5pt; }
  a::after {
    content: " (" attr(href) ")";
    color: #555555;
    font-size: .82em;
    word-break: break-all;
  }
}
"""


COPY_SCRIPT = """
(function () {
  function textFor(target) {
    if (!target) return "";
    if (target.isContentEditable || target.tagName === "TEXTAREA" || target.tagName === "INPUT") {
      return target.value !== undefined && target.value !== "" ? target.value : target.innerText;
    }
    return target.innerText || target.textContent || "";
  }
  document.addEventListener("click", function (event) {
    var btn = event.target.closest("[data-copy-target]");
    if (!btn) return;
    var target = document.getElementById(btn.getAttribute("data-copy-target"));
    var text = textFor(target);
    var done = function () {
      var label = btn.getAttribute("data-copy-label") || btn.textContent;
      btn.setAttribute("data-copy-label", label);
      btn.classList.add("copied");
      btn.textContent = "Copied";
      setTimeout(function () { btn.classList.remove("copied"); btn.textContent = label; }, 1400);
    };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(done, function () { fallbackCopy(text, done); });
    } else {
      fallbackCopy(text, done);
    }
  });
  function fallbackCopy(text, done) {
    var ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand("copy"); done(); } catch (e) {}
    document.body.removeChild(ta);
  }
})();
"""


def slugify(value: str) -> str:
    value = value.strip().lower()
    value = re.sub(r"[^a-z0-9]+", "-", value)
    return value.strip("-") or "report"


def now_iso() -> str:
    return dt.datetime.now(dt.timezone.utc).astimezone().isoformat(timespec="seconds")


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def write_text(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


def parse_frontmatter(text: str) -> tuple[dict[str, str], str]:
    if not text.startswith("---\n"):
        return {}, text
    end = text.find("\n---", 4)
    if end == -1:
        return {}, text
    raw = text[4:end].strip()
    body = text[end + 4 :].lstrip("\n")
    meta: dict[str, str] = {}
    for line in raw.splitlines():
        if ":" not in line:
            continue
        key, value = line.split(":", 1)
        meta[key.strip()] = value.strip().strip('"').strip("'")
    return meta, body


def strip_md(value: str) -> str:
    value = re.sub(r"`([^`]+)`", r"\1", value)
    value = re.sub(r"!\[([^\]]*)\]\([^)]+\)", r"\1", value)
    value = re.sub(r"\[([^\]]+)\]\([^)]+\)", r"\1", value)
    value = re.sub(r"[*_#>]+", "", value)
    return value.strip()


def inline_md(value: str) -> str:
    code_spans: list[str] = []

    def code_repl(match: re.Match[str]) -> str:
        code_spans.append(f"<code>{html.escape(match.group(1))}</code>")
        return f"\x00CODE{len(code_spans) - 1}\x00"

    value = re.sub(r"`([^`]+)`", code_repl, value)
    value = html.escape(value, quote=False)

    def image_repl(match: re.Match[str]) -> str:
        alt = html.escape(match.group(1), quote=True)
        src = html.escape(match.group(2), quote=True)
        return f'<img src="{src}" alt="{alt}" loading="lazy">'

    def link_repl(match: re.Match[str]) -> str:
        label = match.group(1)
        url = html.escape(match.group(2), quote=True)
        return f'<a href="{url}">{label}</a>'

    value = re.sub(r"!\[([^\]]*)\]\(([^)]+)\)", image_repl, value)
    value = re.sub(r"\[([^\]]+)\]\(([^)]+)\)", link_repl, value)
    value = re.sub(r"\*\*([^*]+)\*\*", r"<strong>\1</strong>", value)
    value = re.sub(r"(?<!\*)\*([^*]+)\*(?!\*)", r"<em>\1</em>", value)
    for idx, span in enumerate(code_spans):
        value = value.replace(f"\x00CODE{idx}\x00", span)
    return value


def is_table_delimiter(line: str) -> bool:
    stripped = line.strip()
    if "|" not in stripped:
        return False
    cells = [cell.strip() for cell in stripped.strip("|").split("|")]
    return bool(cells) and all(re.fullmatch(r":?-{3,}:?", cell or "") for cell in cells)


def split_table_row(line: str) -> list[str]:
    return [cell.strip() for cell in line.strip().strip("|").split("|")]


def render_table(lines: list[str]) -> str:
    rows = [split_table_row(line) for line in lines if line.strip()]
    if len(rows) < 2:
        return ""
    header = rows[0]
    body = rows[2:]
    parts = ['<div class="table-scroll"><table><thead><tr>']
    parts.extend(f"<th>{inline_md(cell)}</th>" for cell in header)
    parts.append("</tr></thead><tbody>")
    for row in body:
        parts.append("<tr>")
        for idx in range(len(header)):
            cell = row[idx] if idx < len(row) else ""
            parts.append(f"<td>{inline_md(cell)}</td>")
        parts.append("</tr>")
    parts.append("</tbody></table></div>")
    return "".join(parts)


def is_block_start(line: str, next_line: str | None = None) -> bool:
    stripped = line.strip()
    if not stripped:
        return True
    if stripped.startswith("```"):
        return True
    if re.match(r"^#{1,6}\s+", stripped):
        return True
    if re.match(r"^[-*]\s+", stripped):
        return True
    if re.match(r"^\d+[.)]\s+", stripped):
        return True
    if stripped.startswith(">"):
        return True
    if "|" in stripped and next_line and is_table_delimiter(next_line):
        return True
    return False


def heading_id(text: str, used: set[str]) -> str:
    base = slugify(strip_md(text))
    candidate = base
    counter = 2
    while candidate in used:
        candidate = f"{base}-{counter}"
        counter += 1
    used.add(candidate)
    return candidate


def render_markdown_body(markdown: str) -> tuple[str, list[tuple[str, str]]]:
    lines = markdown.splitlines()
    out: list[str] = []
    nav: list[tuple[str, str]] = []
    used_ids: set[str] = set()
    in_section = False
    in_version_section = False
    version_re = re.compile(r"^Version\s+\d+\b", re.IGNORECASE)

    def close_current_section() -> None:
        nonlocal in_section, in_version_section
        if in_section:
            if in_version_section:
                out.append("</div>")
                in_version_section = False
            out.append("</section>")
            in_section = False

    i = 0
    while i < len(lines):
        line = lines[i]
        stripped = line.strip()
        if not stripped:
            i += 1
            continue

        if stripped.startswith("```"):
            lang = stripped.strip("`").strip()
            i += 1
            code_lines: list[str] = []
            while i < len(lines) and not lines[i].strip().startswith("```"):
                code_lines.append(lines[i])
                i += 1
            if i < len(lines):
                i += 1
            class_attr = f' class="language-{html.escape(lang, quote=True)}"' if lang else ""
            out.append(f"<pre><code{class_attr}>{html.escape(chr(10).join(code_lines))}</code></pre>")
            continue

        heading = re.match(r"^(#{1,6})\s+(.+)$", stripped)
        if heading:
            level = len(heading.group(1))
            text = heading.group(2).strip()
            if level == 2:
                close_current_section()
                sid = heading_id(text, used_ids)
                nav.append((sid, strip_md(text)))
                if version_re.match(strip_md(text)):
                    # Draft version heading (e.g. "Version 1: VE — Value Explainer")
                    # → editable review block with a working copy button.
                    body_id = f"{sid}-draft"
                    out.append(
                        f'<section class="report-section" id="{sid}"><h2>{inline_md(text)}</h2>'
                        f'<div class="draft-actions"><button type="button" class="copy-btn" '
                        f'data-copy-target="{body_id}">Copy draft</button></div>'
                        f'<div class="draft-editable" id="{body_id}" contenteditable="true" spellcheck="true">'
                    )
                    in_section = True
                    in_version_section = True
                else:
                    out.append(f'<section class="report-section" id="{sid}"><h2>{inline_md(text)}</h2>')
                    in_section = True
            elif level == 1:
                out.append(f"<h2>{inline_md(text)}</h2>")
            else:
                tag = f"h{min(level, 4)}"
                out.append(f"<{tag}>{inline_md(text)}</{tag}>")
            i += 1
            continue

        if "|" in stripped and i + 1 < len(lines) and is_table_delimiter(lines[i + 1]):
            table_lines = [lines[i], lines[i + 1]]
            i += 2
            while i < len(lines) and "|" in lines[i] and lines[i].strip():
                table_lines.append(lines[i])
                i += 1
            if not in_section:
                out.append('<section class="report-section">')
                in_section = True
            out.append(render_table(table_lines))
            continue

        if re.match(r"^[-*]\s+", stripped):
            if not in_section:
                out.append('<section class="report-section">')
                in_section = True
            out.append("<ul>")
            while i < len(lines) and re.match(r"^[-*]\s+", lines[i].strip()):
                item = re.sub(r"^[-*]\s+", "", lines[i].strip())
                out.append(f"<li>{inline_md(item)}</li>")
                i += 1
            out.append("</ul>")
            continue

        if re.match(r"^\d+[.)]\s+", stripped):
            if not in_section:
                out.append('<section class="report-section">')
                in_section = True
            out.append("<ol>")
            while i < len(lines) and re.match(r"^\d+[.)]\s+", lines[i].strip()):
                item = re.sub(r"^\d+[.)]\s+", "", lines[i].strip())
                out.append(f"<li>{inline_md(item)}</li>")
                i += 1
            out.append("</ol>")
            continue

        if stripped.startswith(">"):
            if not in_section:
                out.append('<section class="report-section">')
                in_section = True
            quote_lines: list[str] = []
            while i < len(lines) and lines[i].strip().startswith(">"):
                quote_lines.append(lines[i].strip().lstrip(">").strip())
                i += 1
            out.append(f"<blockquote>{inline_md(' '.join(quote_lines))}</blockquote>")
            continue

        paragraph_lines = [stripped]
        i += 1
        while i < len(lines):
            next_line = lines[i]
            following = lines[i + 1] if i + 1 < len(lines) else None
            if is_block_start(next_line, following):
                break
            paragraph_lines.append(next_line.strip())
            i += 1
        if not in_section:
            out.append('<section class="report-section">')
            in_section = True
        klass = ' class="lead-paragraph"' if len(out) < 3 else ""
        out.append(f"<p{klass}>{inline_md(' '.join(paragraph_lines))}</p>")

    close_current_section()
    return "\n".join(out), nav


def infer_subtitle(markdown: str) -> str:
    for line in markdown.splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or stripped.startswith("|"):
            continue
        if stripped.startswith(("-", "*", "```", ">")):
            continue
        return strip_md(stripped)[:220]
    return "A concise intelligence report with evidence, recommendations, and next actions."


def scrub_check(text: str) -> list[str]:
    found: list[str] = []
    for term in CLIENT_BLIND_TERMS:
        if re.search(re.escape(term), text, flags=re.IGNORECASE):
            found.append(term)
    return found


def render_full_html(
    body_html: str,
    nav: list[tuple[str, str]],
    *,
    title: str,
    subtitle: str,
    client_name: str,
    report_kind: str,
    report_date: str,
    generated_at: str,
    status_note: str | None = None,
) -> str:
    nav_html = "\n".join(f'<a href="#{sid}">{html.escape(label)}</a>' for sid, label in nav[:14])
    if not nav_html:
        nav_html = '<a href="#report">Report</a>'
    status = f'<div class="status-note">{inline_md(status_note)}</div>' if status_note else ""
    return f"""<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{html.escape(title)}</title>
<style>{CSS}</style>
</head>
<body>
<div class="report-page">
  <header class="report-hero">
    <div class="hero-copy">
      <p class="eyebrow">{html.escape(report_kind)}</p>
      <h1>{html.escape(title)}</h1>
      <p class="dek">{html.escape(subtitle)}</p>
      {status}
    </div>
    <aside class="hero-panel" aria-label="Report metadata">
      <div class="hero-panel-inner">
        <div class="meta-grid">
          <div class="meta-item"><span class="meta-label">Client</span><span class="meta-value">{html.escape(client_name or "Client")}</span></div>
          <div class="meta-item"><span class="meta-label">Report date</span><span class="meta-value">{html.escape(report_date)}</span></div>
          <div class="meta-item"><span class="meta-label">Format</span><span class="meta-value">HTML report plus PDF companion</span></div>
          <div class="meta-item"><span class="meta-label">Generated</span><span class="meta-value">{html.escape(generated_at)}</span></div>
        </div>
      </div>
    </aside>
  </header>
  <div class="report-shell" id="report">
    <nav class="toc" aria-label="Report sections">
      <p class="toc-title">Report Sections</p>
      {nav_html}
    </nav>
    <main class="report-body">
      {body_html}
    </main>
  </div>
</div>
<script>{COPY_SCRIPT}</script>
</body>
</html>
"""


def find_first_h1(markdown: str) -> str | None:
    match = re.search(r"^#\s+(.+)$", markdown, flags=re.MULTILINE)
    return strip_md(match.group(1)) if match else None


def remove_first_h1(markdown: str) -> str:
    return re.sub(r"^#\s+.+\n?", "", markdown, count=1, flags=re.MULTILINE)


def render_command(args: argparse.Namespace) -> int:
    input_path = Path(args.input)
    markdown_raw = read_text(input_path)
    frontmatter, markdown = parse_frontmatter(markdown_raw)

    title = args.title or frontmatter.get("title") or find_first_h1(markdown) or "Intelligence Report"
    markdown = remove_first_h1(markdown)
    subtitle = args.subtitle or frontmatter.get("subtitle") or infer_subtitle(markdown)
    client_name = args.client_name or frontmatter.get("client") or ""
    report_kind = args.report_kind or frontmatter.get("kind") or "Agency Intelligence Report"
    report_date = args.report_date or frontmatter.get("date") or dt.date.today().isoformat()
    generated_at = now_iso()

    body_html, nav = render_markdown_body(markdown)
    html_text = render_full_html(
        body_html,
        nav,
        title=title,
        subtitle=subtitle,
        client_name=client_name,
        report_kind=report_kind,
        report_date=report_date,
        generated_at=generated_at,
        status_note=args.status_note,
    )

    output_html = Path(args.output_html)
    scrub_found = scrub_check(html_text) if args.client_facing else []
    if args.client_facing and scrub_found and args.fail_on_scrub:
        # Do NOT create/overwrite the real output name on a scrub failure — a
        # downstream step keyed on file existence must not ship a contaminated
        # report. Write to a .blocked.html sidecar and exit 3 instead.
        blocked_html = output_html.with_name(output_html.stem + ".blocked" + output_html.suffix)
        write_text(blocked_html, html_text)
        status = {
            "command": "render",
            "input": str(input_path),
            "output_html": str(output_html),
            "blocked_output_html": str(blocked_html),
            "generated_at": generated_at,
            "client_blind_terms_found": scrub_found,
            "scrub_status": "blocked",
            "pdf_status": "not_generated_scrub_blocked",
            "pdf_path": "",
            "pdf_blocker": "",
        }
        write_status(blocked_html, status)
        print(json.dumps(status, indent=2), file=sys.stderr)
        return 3

    write_text(output_html, html_text)
    status = {
        "command": "render",
        "input": str(input_path),
        "output_html": str(output_html),
        "generated_at": generated_at,
        "client_blind_terms_found": scrub_found,
        "scrub_status": "clean" if args.client_facing else "not_checked",
        "pdf_status": "not_requested",
        "pdf_path": "",
        "pdf_blocker": "",
    }

    if args.output_pdf:
        pdf_status, blocker = export_pdf(output_html, Path(args.output_pdf))
        status["pdf_status"] = pdf_status
        status["pdf_path"] = str(args.output_pdf)
        status["pdf_blocker"] = blocker
    write_status(output_html, status)
    print(json.dumps(status, indent=2))
    return 0 if status["pdf_status"] != "blocked" else 2


def extract_body_fragment(html_text: str) -> str:
    html_text = re.sub(r"<script\b[^>]*>.*?</script>", "", html_text, flags=re.IGNORECASE | re.DOTALL)
    main = re.search(r"<main\b[^>]*>(.*?)</main>", html_text, flags=re.IGNORECASE | re.DOTALL)
    if main:
        return main.group(1).strip()
    body = re.search(r"<body\b[^>]*>(.*?)</body>", html_text, flags=re.IGNORECASE | re.DOTALL)
    if body:
        return body.group(1).strip()
    return html_text


def package_section_label(path: Path) -> str:
    lower_name = path.name.lower()
    if "daily-report" in lower_name:
        return "Daily Cover"
    if "public-data-sources-report" in lower_name:
        return "Public Data Sources"
    if "private-data-sources-report" in lower_name:
        return "Private Data Sources"
    return path.stem.replace("-", " ").title()


def local_href_basename(value: str) -> str:
    parsed = urllib.parse.urlparse(html.unescape(value))
    if parsed.scheme or parsed.netloc:
        return ""
    return Path(urllib.parse.unquote(parsed.path)).name


def rewrite_sibling_report_links(
    fragment: str,
    link_targets: dict[str, tuple[str, str]],
) -> str:
    """Point sibling report-file references at sections inside the package."""

    def href_repl(match: re.Match[str]) -> str:
        quote, value = match.group(1), match.group(2)
        basename = local_href_basename(value)
        if basename in link_targets:
            target, _label = link_targets[basename]
            return f'href={quote}{html.escape(target, quote=True)}{quote}'
        return match.group(0)

    fragment = re.sub(r'\bhref=(["\'])([^"\']+)\1', href_repl, fragment, flags=re.IGNORECASE)

    for basename, (target, label) in link_targets.items():
        escaped_name = html.escape(basename)
        replacement = f'<a href="{html.escape(target, quote=True)}">{html.escape(label)}</a>'

        def cell_repl(match: re.Match[str]) -> str:
            return f"{match.group(1)}{replacement}{match.group(2)}"

        fragment = re.sub(
            rf"(<t[dh][^>]*>\s*){re.escape(escaped_name)}(\s*</t[dh]>)",
            cell_repl,
            fragment,
            flags=re.IGNORECASE,
        )
    return fragment


def namespace_fragment_ids(fragment: str, prefix: str) -> str:
    ids = re.findall(r'\bid=(["\'])([^"\']+)\1', fragment, flags=re.IGNORECASE)
    id_map = {old: f"{prefix}-{old}" for _quote, old in ids if not old.startswith(f"{prefix}-")}
    if not id_map:
        return fragment

    def id_repl(match: re.Match[str]) -> str:
        quote, old = match.group(1), match.group(2)
        return f'id={quote}{html.escape(id_map.get(old, old), quote=True)}{quote}'

    def href_repl(match: re.Match[str]) -> str:
        quote, old = match.group(1), match.group(2)
        return f'href={quote}#{html.escape(id_map.get(old, old), quote=True)}{quote}'

    fragment = re.sub(r'\bid=(["\'])([^"\']+)\1', id_repl, fragment, flags=re.IGNORECASE)
    fragment = re.sub(r'\bhref=(["\'])#([^"\']+)\1', href_repl, fragment, flags=re.IGNORECASE)
    return fragment


def package_command(args: argparse.Namespace) -> int:
    input_paths = [Path(path) for path in args.inputs]
    missing = [str(path) for path in input_paths if not path.exists()]
    generated_at = now_iso()
    sections: list[str] = []
    link_targets = {
        path.name: (f"#part-{idx + 1}", package_section_label(path))
        for idx, path in enumerate(input_paths)
    }
    for idx, path in enumerate(input_paths):
        section_id = f"part-{idx + 1}"
        label = package_section_label(path)
        if not path.exists():
            sections.append(
                f'<section class="report-section" id="{section_id}"><h2>{html.escape(label)}</h2>'
                f"<p>This section was not available when this report was prepared.</p></section>"
            )
            continue
        fragment = extract_body_fragment(read_text(path))
        fragment = rewrite_sibling_report_links(fragment, link_targets)
        fragment = namespace_fragment_ids(fragment, section_id)
        sections.append(
            f'<section class="report-section print-source" id="{section_id}"><div class="print-source-title">'
            f"<strong>{html.escape(label)}</strong></div>{fragment}</section>"
        )

    body_html = "\n".join(sections)
    nav = [(f"part-{idx+1}", package_section_label(Path(path))) for idx, path in enumerate(input_paths)]
    title = args.title or "Client Report"
    html_text = render_full_html(
        body_html,
        nav,
        title=title,
        subtitle=args.subtitle or "A complete client-facing report assembled into one standalone HTML file with a matching PDF companion.",
        client_name=args.client_name or "",
        report_kind=args.report_kind or "Client Report Package",
        report_date=args.report_date or dt.date.today().isoformat(),
        generated_at=generated_at,
        status_note=args.status_note,
    )

    output_html = Path(args.output_html)
    scrub_found = scrub_check(html_text) if args.client_facing else []
    if args.client_facing and scrub_found and args.fail_on_scrub:
        blocked_html = output_html.with_name(output_html.stem + ".blocked" + output_html.suffix)
        write_text(blocked_html, html_text)
        status = {
            "command": "package",
            "inputs": [str(path) for path in input_paths],
            "missing_inputs": missing,
            "output_html": str(output_html),
            "blocked_output_html": str(blocked_html),
            "generated_at": generated_at,
            "client_blind_terms_found": scrub_found,
            "scrub_status": "blocked",
            "pdf_status": "not_generated_scrub_blocked",
            "pdf_path": "",
            "pdf_blocker": "",
        }
        write_status(blocked_html, status)
        print(json.dumps(status, indent=2), file=sys.stderr)
        return 3

    write_text(output_html, html_text)
    status = {
        "command": "package",
        "inputs": [str(path) for path in input_paths],
        "missing_inputs": missing,
        "output_html": str(output_html),
        "generated_at": generated_at,
        "client_blind_terms_found": scrub_found,
        "scrub_status": "clean" if args.client_facing else "not_checked",
        "pdf_status": "not_requested",
        "pdf_path": "",
        "pdf_blocker": "",
    }
    if args.output_pdf:
        pdf_status, blocker = export_pdf(output_html, Path(args.output_pdf))
        status["pdf_status"] = pdf_status
        status["pdf_path"] = str(args.output_pdf)
        status["pdf_blocker"] = blocker
    write_status(output_html, status)
    print(json.dumps(status, indent=2))
    return 0 if status["pdf_status"] != "blocked" else 2


def write_status(output_html: Path, status: dict[str, object]) -> None:
    status_path = output_html.with_suffix(output_html.suffix + ".render_status.json")
    write_text(status_path, json.dumps(status, indent=2, ensure_ascii=False) + "\n")


def chrome_candidates() -> Iterable[str]:
    env = os.environ.get("CHROME_PATH")
    if env:
        yield env
    for name in ("google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "msedge"):
        path = shutil.which(name)
        if path:
            yield path
    mac = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
    if Path(mac).exists():
        yield mac


def export_pdf_with_chrome(html_path: Path, pdf_path: Path) -> tuple[bool, str]:
    errors: list[str] = []
    for chrome in chrome_candidates():
        for headless_flag in ("--headless=new", "--headless"):
            pdf_path.parent.mkdir(parents=True, exist_ok=True)
            with tempfile.TemporaryDirectory(prefix="solo-report-chrome-") as tmp:
                cmd = [
                    chrome,
                    headless_flag,
                    "--disable-gpu",
                    "--disable-dev-shm-usage",
                    "--no-sandbox",
                    "--no-first-run",
                    "--no-default-browser-check",
                    f"--user-data-dir={tmp}",
                    f"--print-to-pdf={pdf_path}",
                    html_path.resolve().as_uri(),
                ]
                try:
                    proc = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, timeout=90)
                except subprocess.TimeoutExpired:
                    errors.append(f"{Path(chrome).name} {headless_flag}: timeout")
                    continue
                if proc.returncode == 0 and pdf_path.exists() and pdf_path.stat().st_size > 0:
                    return True, ""
                last = (proc.stderr or proc.stdout or f"exit {proc.returncode}").strip()[-500:]
                errors.append(f"{Path(chrome).name} {headless_flag}: {last}")
    if errors:
        return False, "chrome_print_failed: " + " ; ".join(errors[:6])
    return False, "chrome_not_found"


def export_pdf_with_weasyprint(html_path: Path, pdf_path: Path) -> tuple[bool, str]:
    try:
        from weasyprint import HTML  # type: ignore
    except Exception as exc:  # pragma: no cover - optional dependency
        return False, f"weasyprint_unavailable: {exc}"
    try:
        pdf_path.parent.mkdir(parents=True, exist_ok=True)
        HTML(filename=str(html_path)).write_pdf(str(pdf_path))
        return True, ""
    except Exception as exc:  # pragma: no cover - optional dependency
        return False, f"weasyprint_failed: {exc}"


class _ReportTextExtractor(HTMLParser):
    block_tags = {
        "address",
        "article",
        "aside",
        "blockquote",
        "br",
        "dd",
        "div",
        "dl",
        "dt",
        "figcaption",
        "figure",
        "footer",
        "h1",
        "h2",
        "h3",
        "h4",
        "h5",
        "h6",
        "header",
        "hr",
        "li",
        "main",
        "ol",
        "p",
        "pre",
        "section",
        "table",
        "tbody",
        "td",
        "tfoot",
        "th",
        "thead",
        "tr",
        "ul",
    }
    heading_tags = {"h1", "h2", "h3"}

    def __init__(self) -> None:
        super().__init__()
        self.blocks: list[tuple[str, str]] = []
        self.parts: list[str] = []
        self.current_style = "body"
        self.skip_depth = 0

    def flush(self) -> None:
        text = re.sub(r"\s+", " ", " ".join(self.parts)).strip()
        if text:
            self.blocks.append((self.current_style, text))
        self.parts = []
        self.current_style = "body"

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        tag = tag.lower()
        if tag in {"script", "style", "noscript", "svg"}:
            self.skip_depth += 1
            return
        if self.skip_depth:
            return
        if tag in self.heading_tags:
            self.flush()
            self.current_style = tag
        elif tag == "li":
            self.flush()
            self.current_style = "li"
            self.parts.append("•")
        elif tag in {"td", "th"}:
            if self.parts:
                self.parts.append("|")
        elif tag in self.block_tags:
            self.flush()

    def handle_endtag(self, tag: str) -> None:
        tag = tag.lower()
        if tag in {"script", "style", "noscript", "svg"} and self.skip_depth:
            self.skip_depth -= 1
            return
        if self.skip_depth:
            return
        if tag in self.block_tags or tag in self.heading_tags:
            self.flush()

    def handle_data(self, data: str) -> None:
        if self.skip_depth:
            return
        if data.strip():
            self.parts.append(data.strip())


def export_pdf_with_reportlab(html_path: Path, pdf_path: Path) -> tuple[bool, str]:
    try:
        from reportlab.lib import colors  # type: ignore
        from reportlab.lib.pagesizes import letter  # type: ignore
        from reportlab.lib.styles import getSampleStyleSheet  # type: ignore
        from reportlab.lib.units import inch  # type: ignore
        from reportlab.platypus import Paragraph, SimpleDocTemplate, Spacer  # type: ignore
    except Exception as exc:  # pragma: no cover - optional dependency
        return False, f"reportlab_unavailable: {exc}"

    try:
        extractor = _ReportTextExtractor()
        extractor.feed(read_text(html_path))
        extractor.flush()

        styles = getSampleStyleSheet()
        styles["Title"].fontName = "Helvetica-Bold"
        styles["Title"].fontSize = 22
        styles["Title"].leading = 27
        styles["Heading1"].fontName = "Helvetica-Bold"
        styles["Heading1"].fontSize = 18
        styles["Heading1"].leading = 22
        styles["Heading2"].fontName = "Helvetica-Bold"
        styles["Heading2"].fontSize = 14
        styles["Heading2"].leading = 18
        styles["BodyText"].fontName = "Helvetica"
        styles["BodyText"].fontSize = 9.5
        styles["BodyText"].leading = 13
        styles["BodyText"].textColor = colors.HexColor("#171512")

        story = []
        for style_name, text in extractor.blocks:
            escaped = html.escape(text)
            if style_name == "h1":
                story.append(Paragraph(escaped, styles["Title"]))
                story.append(Spacer(1, 0.18 * inch))
            elif style_name in {"h2", "h3"}:
                story.append(Paragraph(escaped, styles["Heading1" if style_name == "h2" else "Heading2"]))
                story.append(Spacer(1, 0.09 * inch))
            else:
                story.append(Paragraph(escaped, styles["BodyText"]))
                story.append(Spacer(1, 0.045 * inch))

        if not story:
            return False, "reportlab_failed: no_text_extracted"

        pdf_path.parent.mkdir(parents=True, exist_ok=True)
        doc = SimpleDocTemplate(
            str(pdf_path),
            pagesize=letter,
            leftMargin=0.55 * inch,
            rightMargin=0.55 * inch,
            topMargin=0.55 * inch,
            bottomMargin=0.55 * inch,
            title=html_path.stem,
        )
        doc.build(story)
        return True, ""
    except Exception as exc:  # pragma: no cover - optional dependency
        return False, f"reportlab_failed: {exc}"


def export_pdf_with_wkhtmltopdf(html_path: Path, pdf_path: Path) -> tuple[bool, str]:
    binary = shutil.which("wkhtmltopdf")
    if not binary:
        return False, "wkhtmltopdf_not_found"
    proc = subprocess.run(
        [binary, str(html_path), str(pdf_path)],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        timeout=90,
    )
    if proc.returncode == 0 and pdf_path.exists() and pdf_path.stat().st_size > 0:
        return True, ""
    return False, f"wkhtmltopdf_failed: {(proc.stderr or proc.stdout).strip()[-500:]}"


def export_pdf(html_path: Path, pdf_path: Path) -> tuple[str, str]:
    attempts = [
        ("chrome", export_pdf_with_chrome),
        ("weasyprint", export_pdf_with_weasyprint),
        ("reportlab", export_pdf_with_reportlab),
        ("wkhtmltopdf", export_pdf_with_wkhtmltopdf),
    ]
    blockers: list[str] = []
    for name, attempt in attempts:
        ok, blocker = attempt(html_path, pdf_path)
        if ok:
            # reportlab is a text-only fallback: link URLs, table layout, and print
            # CSS are lost. Flag it so INTERNAL_REPORT can record a degraded PDF.
            return ("generated_degraded" if name == "reportlab" else "generated"), (
                "reportlab_text_only_fallback" if name == "reportlab" else ""
            )
        blockers.append(blocker)
    return "blocked", " | ".join(blockers)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Render OutreachCRM reports to standalone HTML/PDF.")
    subparsers = parser.add_subparsers(dest="command", required=True)

    render = subparsers.add_parser("render", help="Render a Markdown/source report to standalone HTML.")
    render.add_argument("--input", required=True, help="Markdown/source report path.")
    render.add_argument("--output-html", required=True, help="Output HTML path.")
    render.add_argument("--output-pdf", help="Optional output PDF path.")
    render.add_argument("--title", help="Report title. Defaults to first H1 or frontmatter.")
    render.add_argument("--subtitle", help="Hero subtitle.")
    render.add_argument("--client-name", default="", help="Client display name.")
    render.add_argument("--report-kind", default="", help="Daily Report, Public Data Sources Report, etc.")
    render.add_argument("--report-date", default="", help="Report date, YYYY-MM-DD.")
    render.add_argument("--status-note", help="Optional client-safe status note.")
    render.add_argument("--client-facing", action="store_true", help="Run client-blind term scan.")
    render.add_argument("--fail-on-scrub", action="store_true", help="Exit non-zero if client-blind terms are found.")
    render.set_defaults(func=render_command)

    package = subparsers.add_parser("package", help="Package staging HTML files into the single client-facing HTML/PDF.")
    package.add_argument("--inputs", nargs="+", required=True, help="Scrubbed staging HTML inputs, usually daily/public/private.")
    package.add_argument("--output-html", required=True, help="Output package HTML path.")
    package.add_argument("--output-pdf", help="Optional output PDF path.")
    package.add_argument("--title", default="Client Report", help="Package title.")
    package.add_argument("--subtitle", help="Package subtitle.")
    package.add_argument("--client-name", default="", help="Client display name.")
    package.add_argument("--report-kind", default="Client Report Package", help="Package kind label.")
    package.add_argument("--report-date", default="", help="Report date, YYYY-MM-DD.")
    package.add_argument("--status-note", help="Optional client-safe status note.")
    package.add_argument("--client-facing", action="store_true", help="Run client-blind term scan.")
    package.add_argument("--fail-on-scrub", action="store_true", help="Exit non-zero if client-blind terms are found.")
    package.set_defaults(func=package_command)

    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    return int(args.func(args))


if __name__ == "__main__":
    raise SystemExit(main())
