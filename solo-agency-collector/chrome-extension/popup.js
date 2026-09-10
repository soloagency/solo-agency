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
document.getElementById("stopRun").addEventListener("click", stopRun);
  document.getElementById("capture").addEventListener("click", capture);
  document.getElementById("showResult").addEventListener("click", showResult);
  document.getElementById("testScroll").addEventListener("click", testScroll);
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

async function testScroll() {
  setStatus("Testing active-tab scroll: 8 slow page-sized steps. Watch the open tab.");
  const response = await sendMessage({ type: "test_scroll_active_tab", steps: 8, delay_ms: 900 });
  if (!response.ok) {
    setStatus(`Scroll test failed: ${response.error || "unknown error"}`);
    return;
  }
  const lines = (response.debug || []).map((item, index) => {
    return `#${index + 1} ${item.target || "target"} delta=${item.delta}px top=${item.after}px`;
  });
  setStatus([
    "Scroll test finished.",
    `URL: ${response.url || ""}`,
    `Steps: ${response.steps_used || 0}/${response.steps_requested || 0}`,
    ...lines.slice(-8)
  ].join("\n"));
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

async function stopRun() {
  setStatus("Stopping the active run...");
  const response = await sendMessage({ type: "cancel_run", reason: "stopped from the popup", requested_by: "popup" });
  if (!response.ok) {
    setStatus(`Stop failed: ${response.error || "unknown error"}`);
    return;
  }
  const closed = Number(response.tabs_closed || 0);
  setStatus(
    (response.run_id ? `Cancelling run ${response.run_id}. ` : "Cancelling the active run. ") +
    (closed ? `Closed ${closed} collector tab${closed === 1 ? "" : "s"}.` : "No collector tab was open.")
  );
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
  const statusBox = document.getElementById("status");
  if (state.status === "no_client_binding") {
    // The extension has no readable client_binding.json -- the source template folder or a
    // broken client copy was loaded. background.js already refused to poll the bridge; this is
    // just making that unmissable in the popup instead of burying it among the usual status
    // lines. See background.js's NO_CLIENT_BINDING_MESSAGE for the full bilingual sentence.
    if (statusBox) {
      statusBox.style.background = "#fff1f0";
      statusBox.style.borderColor = "#d1242f";
      statusBox.style.color = "#82071e";
    }
    setStatus(state.message || "");
    return;
  }
  if (statusBox) {
    statusBox.style.background = "";
    statusBox.style.borderColor = "";
    statusBox.style.color = "";
  }
  const extensionHealth = state.bridgeStatus && state.bridgeStatus.extension_health
    ? state.bridgeStatus.extension_health
    : null;
  const lines = [
    `Status: ${state.status || "unknown"}`,
    `Message: ${state.message || ""}`,
    planLine(state),
    state.entitlementNote ? `Plan note: ${state.entitlementNote}` : "",
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

// Solo Agency plan as the extension verified it on the last job, else as the bridge reports it.
function planLine(state) {
  const verified = state.entitlement && typeof state.entitlement === "object" ? state.entitlement : null;
  if (verified && verified.tier) {
    return `Plan: ${verified.tier}${verified.verified ? " (verified)" : " (" + (verified.source || "unverified") + ")"}`;
  }
  const bridgeEnt = state.bridgeStatus && state.bridgeStatus.entitlement ? state.bridgeStatus.entitlement : null;
  if (bridgeEnt && bridgeEnt.tier) return `Plan: ${bridgeEnt.tier} (bridge)`;
  return "";
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
