/*
 * audit.js
 * Renders the manual-capture accumulator (chrome.storage.local "collector_audit")
 * so the operator can eyeball the collected text: boilerplate removal, the
 * text(url) link format, tag stripping, and the merged across-scroll output the
 * LLM will receive. Read-only; live-updates as new captures arrive.
 */
const AUDIT_KEY = "collector_audit";

function escapeHTML(value) {
  return String(value || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

// Escape, then highlight URLs so links are easy to scan during audit.
function renderText(text) {
  const escaped = escapeHTML(text);
  return escaped.replace(/https?:\/\/[^\s)<]+/g, (m) => `<span class="u">${m}</span>`);
}

function countUrls(text) {
  const m = String(text || "").match(/https?:\/\/[^\s)]+/g);
  return m ? m.length : 0;
}

function card(label, value, cls) {
  return `<div class="card"><div class="label">${escapeHTML(label)}</div>` +
    `<div class="value${cls ? " " + cls : ""}">${value}</div></div>`;
}

function render(audit) {
  const root = document.getElementById("root");

  if (!audit || !audit.count) {
    root.innerHTML =
      '<div class="empty">No captures yet.<br><br>' +
      'Open a target page, click <b>Capture now</b> in the extension popup, ' +
      'scroll, then capture again.<br>Each capture for the same URL is merged here.</div>';
    return;
  }

  const text = audit.displayText || audit.text || "";
  const chars = text.length;
  const urls = countUrls(text);
  const branch = audit.branch || "?";

  const rows = (audit.captures || []).map((c) =>
    `<tr><td>#${c.n}</td><td>${(c.scrollY || 0).toLocaleString()}px</td>` +
    `<td>${(c.captureChars || 0).toLocaleString()}</td><td>${c.captureUrls || 0}</td>` +
    `<td>${(c.mergedChars || 0).toLocaleString()}</td><td>${c.mergedUrls || 0}</td>` +
    `<td>${c.keptUnits || 0}</td><td>${c.droppedUnits || 0}</td></tr>`
  ).join("");

  root.innerHTML = `
    <header class="top">
      <h1>Collector Audit</h1>
      <span class="badge ${branch === "feed" ? "feed" : "website"}">${escapeHTML(branch)} / ${escapeHTML(audit.platform || "")}</span>
      ${audit.engine ? `<span class="badge">engine: ${escapeHTML(audit.engine)}</span>` : ""}
    </header>
    <div class="sub">${escapeHTML(audit.title || "")}</div>
    <div class="sub">${escapeHTML(audit.url || "")}</div>

    <div class="toolbar">
      <button id="copyBtn">Copy merged text</button>
      <button id="refreshBtn">Refresh</button>
      <span class="hint" id="copyHint"></span>
    </div>

    <div class="cards">
      ${card("Captures", audit.count)}
      ${card("Merged chars", chars.toLocaleString())}
      ${card("Merged urls", urls)}
      ${card("Updated", escapeHTML((audit.updatedAt || "").replace("T", " ").slice(0, 19)), "small")}
    </div>

    <h2>Per-capture (scroll steps)</h2>
    <table>
      <thead><tr><th>Capture</th><th>scrollY</th><th>chars added</th><th>urls in capture</th><th>merged chars</th><th>merged urls</th><th>kept units</th><th>dropped</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>

    <h2>Merged plain text (LLM input)</h2>
    <pre>${renderText(text)}</pre>
  `;

  const copyBtn = document.getElementById("copyBtn");
  if (copyBtn) {
    copyBtn.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(text);
        document.getElementById("copyHint").textContent = "Copied.";
      } catch (e) {
        document.getElementById("copyHint").textContent = "Copy failed: " + e.message;
      }
    });
  }
  const refreshBtn = document.getElementById("refreshBtn");
  if (refreshBtn) refreshBtn.addEventListener("click", load);
}

function load() {
  chrome.storage.local.get(AUDIT_KEY).then((data) => render(data[AUDIT_KEY] || null));
}

load();

// Live update while the page is open and new captures land.
chrome.storage.onChanged.addListener((changes, area) => {
  if (area === "local" && changes[AUDIT_KEY]) {
    render(changes[AUDIT_KEY].newValue || null);
  }
});
