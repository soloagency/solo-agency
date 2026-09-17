// Home tab of the collector window (see collector_window_pool.js). Shows the run the service
// worker is on, straight from collector_state in storage — nothing here talks to the bridge.
(function () {
  const STATE_KEY = "collector_state";
  const badge = document.getElementById("badge");
  const box = document.getElementById("state");

  function render(state) {
    const s = state && typeof state === "object" ? state : {};
    const status = String(s.status || "idle");
    const running = status === "running" || status === "waiting_for_operator";
    badge.textContent = status.replace(/_/g, " ");
    badge.className = "badge" + (running ? " running" : "");
    const lines = [];
    if (s.message) lines.push(escapeHtml(String(s.message)));
    if (s.runId) lines.push('Run <code>' + escapeHtml(String(s.runId)) + '</code>');
    if (typeof s.sourcesDone === "number" && typeof s.totalSources === "number") {
      lines.push("Sources: " + s.sourcesDone + " / " + s.totalSources);
    }
    if (s.currentSource) lines.push("Current: " + escapeHtml(String(s.currentSource)));
    if (s.updatedAt) lines.push('<span class="muted">Updated ' + escapeHtml(String(s.updatedAt)) + "</span>");
    box.innerHTML = lines.length ? lines.join("<br>") : '<span class="muted">No run in progress.</span>';
  }

  function escapeHtml(text) {
    return text.replace(/[&<>"']/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
    });
  }

  function load() {
    try {
      chrome.storage.local.get(STATE_KEY, function (data) { render(data && data[STATE_KEY]); });
    } catch (error) { render(null); }
  }

  try {
    chrome.storage.onChanged.addListener(function (changes, area) {
      if (area === "local" && changes[STATE_KEY]) render(changes[STATE_KEY].newValue);
    });
  } catch (error) { /* no storage events: the page still shows the state it loaded */ }
  load();
})();
