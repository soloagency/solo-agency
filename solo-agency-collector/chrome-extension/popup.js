const fields = [
  "enabled",
  "bridgeBaseUrl",
  "pollMinutes",
  "pollSeconds",
  "minDelaySeconds",
  "maxDelaySeconds",
  "maxSourcesPerRun",
  "sourceConcurrency",
  "scrollSteps",
  "closeTabsAfterCollect"
];

document.addEventListener("DOMContentLoaded", async () => {
  await refresh();
  document.getElementById("save").addEventListener("click", save);
  document.getElementById("checkNow").addEventListener("click", checkNow);
  document.getElementById("capture").addEventListener("click", capture);
  document.getElementById("showResult").addEventListener("click", showResult);
  document.getElementById("resetAudit").addEventListener("click", resetAudit);
});

async function refresh() {
  const response = await sendMessage({ type: "get_state" });
  if (!response.ok) {
    setStatus(`Error: ${response.error || "Unable to load state."}`);
    return;
  }
  fillSettings(response.settings || {});
  renderState(response.state || {});
  const audit = await sendMessage({ type: "get_audit" });
  if (audit && audit.ok) renderAudit(audit.audit);
}

async function capture() {
  setStatus("Capturing active tab...");
  const response = await sendMessage({ type: "manual_capture" });
  if (!response.ok) {
    setStatus(`Capture failed: ${response.error || "unknown error"}`);
    return;
  }
  renderAudit(response.audit);
  setStatus(`Captured #${response.audit.count} (${response.audit.branch}/${response.audit.platform}).`);
}

function showResult() {
  chrome.tabs.create({ url: chrome.runtime.getURL("audit.html") });
}

async function resetAudit() {
  await sendMessage({ type: "reset_audit" });
  renderAudit(null);
  setStatus("Captures reset.");
}

function renderAudit(audit) {
  const el = document.getElementById("auditInfo");
  if (!el) return;
  if (!audit || !audit.count) {
    el.textContent = "No captures yet.";
    return;
  }
  const shown = audit.displayText || audit.text || "";
  const chars = shown.length;
  const urls = (shown.match(/https?:\/\/[^\s)]+/g) || []).length;
  const eng = audit.engine ? ` • engine:${audit.engine}` : "";
  el.textContent = `Captures: ${audit.count} • ${audit.branch}/${audit.platform}${eng} • ${chars} chars • ${urls} urls — click "Show result" to audit.`;
}

async function save() {
  const settings = readSettings();
  const response = await sendMessage({ type: "save_settings", settings });
  if (!response.ok) {
    setStatus(`Save failed: ${response.error || "unknown error"}`);
    return;
  }
  await refresh();
}

async function checkNow() {
  setStatus("Checking local bridge...");
  const response = await sendMessage({ type: "check_now" });
  if (!response.ok) {
    setStatus(`Check failed: ${response.error || "unknown error"}`);
    return;
  }
  await refresh();
}

function fillSettings(settings) {
  for (const field of fields) {
    const el = document.getElementById(field);
    if (!el) continue;
    if (el.type === "checkbox") {
      el.checked = Boolean(settings[field]);
    } else if (settings[field] !== undefined) {
      el.value = settings[field];
    }
  }
}

function readSettings() {
  return {
    enabled: document.getElementById("enabled").checked,
    bridgeBaseUrl: document.getElementById("bridgeBaseUrl").value.trim(),
    pollMinutes: Number(document.getElementById("pollMinutes").value),
    pollSeconds: Number(document.getElementById("pollSeconds").value),
    minDelaySeconds: Number(document.getElementById("minDelaySeconds").value),
    maxDelaySeconds: Number(document.getElementById("maxDelaySeconds").value),
    maxSourcesPerRun: Number(document.getElementById("maxSourcesPerRun").value),
    sourceConcurrency: Number(document.getElementById("sourceConcurrency").value),
    scrollSteps: Number(document.getElementById("scrollSteps").value),
    closeTabsAfterCollect: document.getElementById("closeTabsAfterCollect").checked
  };
}

function renderState(state) {
  const extensionHealth = state.bridgeStatus && state.bridgeStatus.extension_health
    ? state.bridgeStatus.extension_health
    : null;
  const lines = [
    `Status: ${state.status || "unknown"}`,
    `Message: ${state.message || ""}`,
    state.lastBridgeContactAt ? `Last bridge contact: ${state.lastBridgeContactAt}` : "",
    extensionHealth ? `Bridge sees extension: ${extensionHealth.status || "unknown"}` : "",
    extensionHealth && extensionHealth.last_extension_check_at ? `Last extension check: ${extensionHealth.last_extension_check_at}` : "",
    extensionHealth && extensionHealth.extension_check_count !== undefined ? `Extension checks: ${extensionHealth.extension_check_count}` : "",
    state.runId ? `Run: ${state.runId}` : "",
    state.currentSource ? `Current source: ${state.currentSource}` : "",
    state.maxScrolls !== undefined ? `Scroll: ${state.currentScroll || 0}/${state.maxScrolls || 0}` : "",
    state.dataPointsCollected !== undefined ? `Data points: ${state.dataPointsCollected}` : "",
    state.competitorsDetected !== undefined ? `Competitors: ${state.competitorsDetected}` : "",
    state.newPrivateSourcesDetected !== undefined ? `New private sources: ${state.newPrivateSourcesDetected}` : "",
    state.updatedAt ? `Updated: ${state.updatedAt}` : ""
  ].filter(Boolean);
  setStatus(lines.join("\n"));
}

function setStatus(text) {
  document.getElementById("status").textContent = text;
}

function sendMessage(message) {
  return new Promise((resolve) => {
    chrome.runtime.sendMessage(message, (response) => {
      const err = chrome.runtime.lastError;
      if (err) {
        resolve({ ok: false, error: err.message });
      } else {
        resolve(response || { ok: false, error: "empty response" });
      }
    });
  });
}
