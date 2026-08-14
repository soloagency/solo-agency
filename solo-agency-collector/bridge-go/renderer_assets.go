package main

// renderer_assets.go — CSS + copy-button script, extracted verbatim from
// outreach/tools/report_renderer.py so the rendered HTML stays byte-identical.

const rendererCSS = `
:root {
  color-scheme: dark;
  --ink: #070a08;
  --surface: #0d1210;
  --surface-2: #121814;
  --line: rgba(242, 245, 251, 0.09);
  --line-strong: rgba(166, 220, 22, 0.28);
  --text: #eef2ee;
  --text-soft: #c6d0c8;
  --muted: #94a398;
  --accent: #a6dc16;
  --accent-dim: rgba(166, 220, 22, 0.12);
  --accent-ink: #070a08;
  --r: 12px;
  --font: "Avenir Next", Avenir, "Helvetica Neue", Helvetica, Arial, sans-serif;
}
* { box-sizing: border-box; }
html, body { margin: 0; padding: 0; }
body {
  background: var(--ink);
  color: var(--text);
  font-family: var(--font);
  font-size: 16px;
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;
}
.report-page { width: min(100%, 1180px); margin: 0 auto; padding: 40px 24px 96px; }

/* ---------- hero ---------- */
.report-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.6fr) minmax(260px, 1fr);
  gap: 32px;
  align-items: end;
  padding: 24px 0 40px;
  border-bottom: 1px solid var(--line);
  margin-bottom: 40px;
}
.eyebrow {
  color: var(--accent);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  margin: 0 0 12px;
}
h1 {
  font-size: clamp(1.9rem, 1.2rem + 2.4vw, 3rem);
  line-height: 1.12;
  font-weight: 700;
  letter-spacing: -0.015em;
  margin: 0 0 12px;
  text-wrap: balance;
}
.dek { color: var(--text-soft); font-size: 1.05rem; max-width: 58ch; margin: 0; }
.status-note {
  display: inline-block;
  margin-top: 16px;
  padding: 6px 12px;
  border: 1px solid var(--line-strong);
  border-radius: 999px;
  background: var(--accent-dim);
  color: var(--accent);
  font-size: 0.85rem;
  font-weight: 600;
}
.hero-panel {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--r);
  overflow: hidden;
}
.hero-panel::before { content: ""; display: block; height: 3px; background: var(--accent); }
.hero-panel-inner { padding: 18px 20px; }
.meta-grid { display: grid; gap: 10px; }
.meta-item {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--line);
}
.meta-item:last-child { border-bottom: 0; padding-bottom: 0; }
.meta-label { color: var(--muted); font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.08em; }
.meta-value { font-size: 0.9rem; font-weight: 600; text-align: right; font-variant-numeric: tabular-nums; }

/* ---------- shell: toc + body ---------- */
.report-shell { display: grid; grid-template-columns: 208px minmax(0, 1fr); gap: 40px; }
.toc { position: sticky; top: 24px; align-self: start; }
.toc-title { color: var(--muted); font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.12em; margin: 0 0 10px; }
.toc a {
  display: block;
  color: var(--text-soft);
  text-decoration: none;
  font-size: 0.88rem;
  padding: 6px 10px;
  border-left: 2px solid var(--line);
  transition: color 0.2s, border-color 0.2s;
}
.toc a:hover { color: var(--accent); border-left-color: var(--accent); }
.report-body { min-width: 0; }

/* ---------- sections as cards ---------- */
.report-section {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--r);
  padding: 26px 28px;
  margin: 0 0 24px;
}
.report-section h2 {
  font-size: 1.3rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  line-height: 1.25;
  margin: 0 0 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--line);
}
.report-section h3 { font-size: 1.05rem; font-weight: 650; margin: 22px 0 8px; color: var(--text); }
.report-section h4 { font-size: 0.92rem; font-weight: 600; margin: 16px 0 6px; color: var(--text-soft); }
.report-section p { margin: 0 0 12px; max-width: 72ch; }
.report-section > p:first-child { margin-top: 0; }
.lead-paragraph { font-size: 1.08rem; color: var(--text-soft); }
ul, ol { margin: 0 0 14px; padding-left: 22px; }
li { margin: 4px 0; }
li::marker { color: var(--accent); }
strong { color: var(--text); font-weight: 650; }
a { color: var(--accent); text-decoration: underline; text-decoration-color: var(--line-strong); text-underline-offset: 3px; }
a:hover { text-decoration-color: var(--accent); }
a::after { content: ""; }
blockquote {
  margin: 14px 0;
  padding: 12px 18px;
  border-left: 3px solid var(--accent);
  background: var(--accent-dim);
  border-radius: 0 10px 10px 0;
  color: var(--text-soft);
}
blockquote p:last-child { margin-bottom: 0; }
hr { border: 0; border-top: 1px solid var(--line); margin: 24px 0; }

/* ---------- tables ---------- */
.table-scroll { overflow-x: auto; margin: 14px 0; border: 1px solid var(--line); border-radius: 10px; }
table { border-collapse: collapse; width: 100%; font-size: 0.92rem; }
th, td { padding: 10px 14px; text-align: left; border-bottom: 1px solid var(--line); vertical-align: top; }
th {
  background: var(--surface-2);
  color: var(--muted);
  font-size: 0.78rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.07em;
  white-space: nowrap;
}
td { font-variant-numeric: tabular-nums; }
tr:last-child td { border-bottom: 0; }

/* ---------- code ---------- */
code {
  font-family: ui-monospace, "SF Mono", Menlo, Consolas, monospace;
  font-size: 0.86em;
  background: var(--surface-2);
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 1px 6px;
}
pre {
  background: var(--surface-2);
  border: 1px solid var(--line);
  border-radius: 10px;
  padding: 14px 16px;
  overflow-x: auto;
  margin: 14px 0;
}
pre code { background: none; border: 0; padding: 0; font-size: 0.85rem; }

/* ---------- draft blocks (editable + copy) ---------- */
.draft-actions { display: flex; justify-content: flex-end; margin: 0 0 8px; }
.copy-btn {
  font-family: var(--font);
  font-size: 0.8rem;
  font-weight: 650;
  color: var(--accent-ink);
  background: var(--accent);
  border: 0;
  border-radius: 8px;
  padding: 6px 14px;
  cursor: pointer;
}
.copy-btn:hover { filter: brightness(1.08); }
.copy-btn:active { transform: translateY(1px); }
.copy-btn.copied { background: var(--surface-2); color: var(--accent); border: 1px solid var(--line-strong); }
.draft-editable {
  border: 1px dashed var(--line-strong);
  border-radius: 10px;
  padding: 14px 16px;
  background: var(--surface-2);
}
.draft-editable:focus { outline: 2px solid var(--accent); outline-offset: 2px; }
.print-source-title { color: var(--muted); font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.1em; margin-bottom: 8px; }

/* ---------- responsive ---------- */
@media (max-width: 900px) {
  .report-hero { grid-template-columns: 1fr; align-items: start; }
  .report-shell { grid-template-columns: 1fr; gap: 16px; }
  .toc { position: static; display: flex; flex-wrap: wrap; gap: 6px; }
  .toc-title { flex-basis: 100%; }
  .toc a { border: 1px solid var(--line); border-radius: 999px; padding: 5px 12px; }
  .report-section { padding: 20px 18px; }
}

/* ---------- print: the PDF companion flips to paper ---------- */
@media print {
  :root { color-scheme: light; }
  body { background: #ffffff; color: #17201a; font-size: 12px; }
  .report-page { padding: 0; width: 100%; }
  .report-hero { border-bottom-color: #d8ded9; }
  .eyebrow, .toc a:hover, li::marker { color: #4a7005; }
  .dek, .lead-paragraph, .meta-label, .toc-title, .print-source-title { color: #556058; }
  .status-note { background: #f1f7e3; border-color: #b9d67a; color: #3d5c04; }
  .hero-panel, .report-section, .table-scroll, .draft-editable, pre, code {
    background: #ffffff;
    border-color: #d8ded9;
  }
  .hero-panel::before { background: #a6dc16; }
  .meta-value, strong, .report-section h3 { color: #17201a; }
  .report-section { break-inside: avoid; box-shadow: none; }
  .report-section h2 { border-bottom-color: #d8ded9; }
  th { background: #f4f6f2; color: #556058; }
  th, td { border-bottom-color: #e2e7e2; }
  a { color: #3d5c04; text-decoration-color: #b9d67a; }
  blockquote { background: #f6faeb; border-left-color: #a6dc16; color: #333d35; }
  .toc, .draft-actions, .copy-btn { display: none; }
  .draft-editable { border-style: solid; }
}
`

const rendererCopyScript = `
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
`
