package main

// renderer_assets.go — CSS + copy-button script, extracted verbatim from
// outreach/tools/report_renderer.py so the rendered HTML stays byte-identical.

const rendererCSS = `
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
