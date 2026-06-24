const DEFAULT_SETTINGS = {
  enabled: true,
  bridgeBaseUrl: "http://127.0.0.1:17321",
  pollMinutes: 1,
  pollSeconds: 5,
  minDelaySeconds: 5,
  maxDelaySeconds: 5,
  maxSourcesPerRun: 20,
  sourceConcurrency: 1,
  scrollSteps: 5,
  maxTextChars: 12000,
  closeTabsAfterCollect: true
};

const STATE_KEY = "collector_state";
const SETTINGS_KEY = "collector_settings";
const COMPLETED_RUNS_KEY = "collector_completed_runs";
const ACTIVE_RUN_KEY = "collector_active_run";
const AUDIT_KEY = "collector_audit";
const CAPTURE_FILES = ["collector_helpers.js", "readability.js", "filtering.js", "infinity_loops.js"];
const ACTIVE_RUN_LOCK_MINUTES = 120;
const EXTENSION_BUILD = "0.1.5-humanized-scroll";
const NORMAL_SCROLL_CAP = 10;
const DISCOVERY_SCROLL_CAP = 80;

const inMemoryActiveRuns = new Set();
let clientBindingCache = null;

chrome.runtime.onInstalled.addListener(async () => {
  const settings = await getSettings();
  await chrome.storage.local.set({ [SETTINGS_KEY]: settings });
  chrome.alarms.create("collector_poll", { periodInMinutes: Math.max(1, settings.pollMinutes || 1) });
  scheduleShortPoll(settings);
  await setState({ status: "installed", message: "Collector installed and idle." });
  await pollBridge("installed");
});

chrome.runtime.onStartup.addListener(async () => {
  const settings = await getSettings();
  chrome.alarms.create("collector_poll", { periodInMinutes: Math.max(1, settings.pollMinutes || 1) });
  scheduleShortPoll(settings);
  await pollBridge("startup");
});

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === "collector_poll") {
    pollBridge("alarm").catch((error) => setState({
      status: "error",
      message: String(error && error.message ? error.message : error),
      updatedAt: new Date().toISOString()
    }));
  }
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  (async () => {
    if (message && message.type === "get_state") {
      sendResponse({ ok: true, state: await getState(), settings: await getSettings() });
      return;
    }
    if (message && message.type === "save_settings") {
      const next = normalizeSettings(message.settings || {});
      await chrome.storage.local.set({ [SETTINGS_KEY]: next });
      chrome.alarms.create("collector_poll", { periodInMinutes: Math.max(1, next.pollMinutes || 1) });
      scheduleShortPoll(next);
      const bridgeConfigSaved = await syncSettingsToBridge(next);
      await setState({
        status: "settings_saved",
        message: bridgeConfigSaved === true
          ? "Settings saved locally and to bridge."
          : bridgeConfigSaved === "skipped"
            ? "Settings saved locally. Shared agency config is managed by the agent and bridge."
            : "Settings saved locally. Bridge config was not updated.",
        bridgeConfigSaved
      });
      const pollResult = await pollBridge("settings_saved");
      sendResponse({ ok: true, settings: next, bridgeConfigSaved, pollResult });
      return;
    }
    if (message && message.type === "check_now") {
      const result = await pollBridge("manual");
      sendResponse({ ok: true, result, state: await getState() });
      return;
    }
    if (message && message.type === "manual_capture") {
      sendResponse(await manualCapture());
      return;
    }
    if (message && message.type === "test_scroll_active_tab") {
      sendResponse(await testScrollActiveTab(message));
      return;
    }
    if (message && message.type === "get_audit") {
      sendResponse({ ok: true, audit: await getAudit() });
      return;
    }
    if (message && message.type === "reset_audit") {
      await chrome.storage.local.remove(AUDIT_KEY);
      sendResponse({ ok: true });
      return;
    }
    sendResponse({ ok: false, error: "unknown message" });
  })().catch((error) => sendResponse({ ok: false, error: String(error && error.message ? error.message : error) }));
  return true;
});

let shortPollTimer = null;

function scheduleShortPoll(settings) {
  if (shortPollTimer) clearTimeout(shortPollTimer);
  if (!settings || !settings.enabled) return;
  const seconds = clampNumber(settings.pollSeconds, 5, 60, 5);
  shortPollTimer = setTimeout(async () => {
    try {
      await pollBridge("short_poll");
    } finally {
      scheduleShortPoll(await getSettings());
    }
  }, seconds * 1000);
}

async function pollBridge(reason) {
  const settings = await getSettings();
  const binding = await getClientBinding();
  if (!settings.enabled) {
    await setState({ status: "disabled", message: "Collector disabled.", reason });
    return { status: "disabled" };
  }

  const bridgeBaseUrl = trimSlash(settings.bridgeBaseUrl);
  let status;
  try {
    status = await fetchJSON(`${bridgeBaseUrl}/status`, {
      method: "GET",
      headers: collectorHeaders(binding)
    }, 6000);
  } catch (error) {
    await setState({
      status: "bridge_offline",
      message: "Local bridge is not running.",
      bridgeBaseUrl,
      reason,
      updatedAt: new Date().toISOString()
    });
    return { status: "bridge_offline" };
  }
  const bridgeContactAt = new Date().toISOString();

  if (!status || !status.job_available || status.completed) {
    await setState({
      status: "idle",
      message: "Bridge is online but no unfinished job is available.",
      bridgeStatus: status,
      lastBridgeContactAt: bridgeContactAt,
      reason,
      updatedAt: new Date().toISOString()
    });
    return { status: "idle" };
  }

  const job = await fetchJSON(`${bridgeBaseUrl}/jobs/current`, {
    method: "GET",
    headers: collectorHeaders(binding)
  }, 8000);
  if (!jobMatchesBinding(job, binding)) {
    await setState({
      status: "job_for_other_client",
      message: "Bridge returned a job for a different client or extension.",
      runId: job && job.run_id ? String(job.run_id) : "",
      bridgeStatus: status,
      lastBridgeContactAt: bridgeContactAt,
      reason,
      updatedAt: new Date().toISOString()
    });
    return { status: "job_for_other_client" };
  }
  const session = job.collector_bridge || {};
  const runId = String(job.run_id || session.run_id || status.run_id || "");
  if (!runId) {
    await setState({ status: "invalid_job", message: "Collector job has no run_id.", reason });
    return { status: "invalid_job" };
  }

  const completedRuns = await getCompletedRuns();
  if (completedRuns[runId] && !job.force) {
    await setState({
      status: "already_completed",
      message: `Run ${runId} was already completed by this extension.`,
      runId,
      bridgeStatus: status,
      lastBridgeContactAt: bridgeContactAt,
      reason,
      updatedAt: new Date().toISOString()
    });
    return { status: "already_completed", runId };
  }

  const runLock = await acquireRunLock(runId, job, status, reason);
  if (!runLock.acquired) {
    await setState({
      status: "already_running",
      message: `Run ${runId} is already being collected. Ignoring ${reason} poll.`,
      runId,
      lockReason: runLock.reason,
      activeLock: runLock.lock || null,
      bridgeStatus: status,
      lastBridgeContactAt: bridgeContactAt,
      reason,
      updatedAt: new Date().toISOString()
    });
    return { status: "already_running", runId, reason: runLock.reason };
  }

  const token = session.write_token;
  if (!token) {
    await releaseRunLock(runId, runLock.owner);
    await setState({ status: "invalid_job", message: "Collector job missing write token.", runId, reason });
    return { status: "invalid_job" };
  }

  try {
    await runJob({ job, token, bridgeBaseUrl, settings, binding, reason });
    completedRuns[runId] = new Date().toISOString();
    await chrome.storage.local.set({ [COMPLETED_RUNS_KEY]: trimCompletedRuns(completedRuns) });
    return { status: "completed", runId };
  } finally {
    await releaseRunLock(runId, runLock.owner);
  }
}

async function runJob({ job, token, bridgeBaseUrl, settings, binding, reason }) {
  const runId = String(job.run_id || job.collector_bridge?.run_id || "");
  const sources = normalizeSources(job.sources || []);
  const pacing = job.pacing || {};
  const maxSources = Math.min(
    Number(pacing.max_sources || settings.maxSourcesPerRun || 20),
    sources.length
  );
  const selectedSources = sources.slice(0, maxSources);
  const sourceConcurrency = Math.min(
    selectedSources.length || 1,
    clampNumber(
      pacing.source_concurrency || pacing.max_parallel_sources || settings.sourceConcurrency,
      1,
      3,
      1
    )
  );
  const counts = {
    dataPoints: 0,
    competitors: 0,
    newPrivateSources: 0
  };

  await setState({
    status: "running",
    message: `Collecting ${selectedSources.length} private sources with ${sourceConcurrency} tab${sourceConcurrency === 1 ? "" : "s"}.`,
    runId,
    totalSources: selectedSources.length,
    sourceConcurrency,
    dataPointsCollected: counts.dataPoints,
    competitorsDetected: counts.competitors,
    newPrivateSourcesDetected: counts.newPrivateSources,
    reason,
    updatedAt: new Date().toISOString()
  });

  async function processSource(source, index) {
    const sourceLabel = source.name || source.url || `source ${index + 1}`;
    await postToBridge(bridgeBaseUrl, token, "/collect/source_status", {
      run_id: runId,
      client_slug: job.client_slug || binding.client_slug || "",
      source_name: source.name || "",
      source_url: source.url || "",
      platform: source.platform || "",
      status: "started",
      index: index + 1,
      total: selectedSources.length,
      source_concurrency: sourceConcurrency,
      captured_at: new Date().toISOString(),
      collector_identity: "chrome-extension-local-collector"
    }, binding);

    await setState({
      status: "running",
      message: `Collecting ${index + 1}/${selectedSources.length}: ${sourceLabel}`,
      runId,
      currentSource: sourceLabel,
      sourceConcurrency,
      currentScroll: 0,
      maxScrolls: Number(job.pacing?.scroll_steps || settings.scrollSteps || 5),
      dataPointsCollected: counts.dataPoints,
      competitorsDetected: counts.competitors,
      newPrivateSourcesDetected: counts.newPrivateSources,
      updatedAt: new Date().toISOString()
    });

    await delay(randomDelayMs(settings, pacing));

    try {
      const collected = await collectSource(source, job, settings, binding, index + 1);
      await postToBridge(bridgeBaseUrl, token, "/collect/data_point", collected.dataPoint, binding);
      counts.dataPoints += 1;
      if (isCompetitorSource(source)) {
        await postToBridge(bridgeBaseUrl, token, "/collect/competitor", {
          run_id: runId,
          client_slug: job.client_slug || binding.client_slug || "",
          source_name: source.name || "",
          platform: source.platform || "",
          competitor_type: source.competitor_type || "unknown",
          name_or_page: source.name || collected.dataPoint.title || "",
          profile_url: collected.dataPoint.profile_url || source.url || "unavailable",
          post_url: collected.dataPoint.post_url || collected.dataPoint.current_url || "unavailable",
          current_url: collected.dataPoint.current_url || "",
          location_relevance: source.location_relevance || "",
          audience_overlap: source.audience_overlap || "",
          offer_positioning: "",
          content_themes: "",
          engagement_signal: collected.dataPoint.engagement_hint || "",
          threat_level: "unknown",
          opportunity: "AI agent should review this competitor source and derive positioning opportunities.",
          captured_at: new Date().toISOString(),
          collector_identity: "chrome-extension-local-collector"
        }, binding);
        counts.competitors += 1;
      }
      for (const candidate of collected.newPrivateSources || []) {
        await postToBridge(bridgeBaseUrl, token, "/collect/new_private_source", candidate, binding);
        counts.newPrivateSources += 1;
      }
      await postToBridge(bridgeBaseUrl, token, "/collect/snapshot", collected.snapshot, binding);
      await postToBridge(bridgeBaseUrl, token, "/collect/source_status", {
        run_id: runId,
        client_slug: job.client_slug || binding.client_slug || "",
        source_name: source.name || "",
        source_url: source.url || "",
        platform: source.platform || "",
        status: "collected",
        index: index + 1,
        total: selectedSources.length,
        source_concurrency: sourceConcurrency,
        current_url: collected.dataPoint.current_url || "",
        post_url: collected.dataPoint.post_url || collected.dataPoint.current_url || "",
        profile_url: collected.dataPoint.profile_url || "",
        captured_at: new Date().toISOString(),
        collector_identity: "chrome-extension-local-collector"
      }, binding);
      await setState({
        status: "running",
        message: `Collected ${index + 1}/${selectedSources.length}: ${sourceLabel}`,
        runId,
        currentSource: sourceLabel,
        sourceConcurrency,
        currentScroll: collected.dataPoint.scroll_count || 0,
        maxScrolls: collected.dataPoint.max_scrolls || 0,
        dataPointsCollected: counts.dataPoints,
        competitorsDetected: counts.competitors,
        newPrivateSourcesDetected: counts.newPrivateSources,
        updatedAt: new Date().toISOString()
      });
    } catch (error) {
      await postToBridge(bridgeBaseUrl, token, "/collect/source_status", {
        run_id: runId,
        client_slug: job.client_slug || binding.client_slug || "",
        source_name: source.name || "",
        source_url: source.url || "",
        platform: source.platform || "",
        status: "error",
        issue: String(error && error.message ? error.message : error),
        index: index + 1,
        total: selectedSources.length,
        source_concurrency: sourceConcurrency,
        captured_at: new Date().toISOString(),
        collector_identity: "chrome-extension-local-collector"
      }, binding);
    }
  }

  let nextIndex = 0;
  const workerCount = Math.max(1, sourceConcurrency);
  const workers = Array.from({ length: workerCount }, async () => {
    while (nextIndex < selectedSources.length) {
      const index = nextIndex;
      nextIndex += 1;
      const source = selectedSources[index];
      await processSource(source, index);
    }
  });
  await Promise.all(workers);

  await postToBridge(bridgeBaseUrl, token, "/complete", {
    run_id: runId,
    client_slug: job.client_slug || binding.client_slug || "",
    status: "completed",
    completed_at: new Date().toISOString(),
    collector_identity: "chrome-extension-local-collector"
  }, binding);

  await setState({
    status: "completed",
    message: `Completed run ${runId}.`,
    runId,
    totalSources: selectedSources.length,
    dataPointsCollected: counts.dataPoints,
    competitorsDetected: counts.competitors,
    newPrivateSourcesDetected: counts.newPrivateSources,
    updatedAt: new Date().toISOString()
  });
}

async function collectSource(source, job, settings, binding, sourceIndex) {
  if (!source.url || !/^https?:\/\//i.test(source.url)) {
    throw new Error("source url must start with http:// or https://");
  }

  const tab = await createTab({ url: source.url, active: false });
  try {
    await waitForTabLoad(tab.id, 25000);
    await delay(randomDelayMs(settings, job.pacing || {}));
    // Inject the cleaning pipeline (filtering.js + helpers), then run the
    // scroll/capture orchestrator. Same data_point envelope as before; the text
    // fields now carry the cleaned extraction instead of the raw innerText blob.
    await chrome.scripting.executeScript({ target: { tabId: tab.id }, files: CAPTURE_FILES });
    const discoveryMode = isDiscoveryCollection(job, source);
    const maxScrollSteps = maxScrollStepsForCollection(job, source);
    const [result] = await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: collectCleanPage,
      args: [{
        scrollSteps: Number(job.pacing?.scroll_steps || settings.scrollSteps || 5),
        maxScrollSteps,
        scrollMode: discoveryMode ? "discovery" : "standard",
        stopAfterNoMoveScrolls: discoveryMode ? 3 : 0,
        minDelaySeconds: Number(job.pacing?.min_delay_seconds || settings.minDelaySeconds || 5),
        maxDelaySeconds: Number(job.pacing?.max_delay_seconds || settings.maxDelaySeconds || 5)
      }]
    });

    const cap = result && result.result ? result.result : {};
    const now = new Date().toISOString();
    const runId = String(job.run_id || "");
    const maxChars = Number(job.pacing?.max_text_chars || settings.maxTextChars || 12000);
    const text = String(cap.mergedDisplay || cap.merged || "");
    const excerpt = text.slice(0, maxChars);
    const accountUrls = Array.isArray(cap.accountUrls) ? cap.accountUrls : [];
    const postUrls = Array.isArray(cap.postUrls) ? cap.postUrls : [];
    const entityItems = Array.isArray(cap.entityItems) ? cap.entityItems : [];
    const entityProfileUrls = entityItems
      .filter((it) => it && /profile|account/i.test(String(it.type || "")))
      .map((it) => it.url)
      .filter(Boolean);
    const profileCandidates = [];
    const seenProfileCandidates = new Set();
    for (const url of accountUrls.concat(entityProfileUrls)) {
      const key = canonicalSourceKey(url);
      if (!key || seenProfileCandidates.has(key)) continue;
      seenProfileCandidates.add(key);
      profileCandidates.push(url);
    }
    const pageLike = {
      current_url: cap.url || "",
      title: cap.title || "",
      profile_candidates: profileCandidates,
      post_candidates: postUrls,
      entity_candidates: entityItems,
      raw_visible_text_excerpt: excerpt
    };
    const snapshotNonce = Math.random().toString(36).slice(2, 8);
    const snapshotFilename = `${safeSlug(runId || "run")}_${String(sourceIndex || 0).padStart(2, "0")}_${safeSlug(source.name || source.platform || "source")}_${Date.now()}_${snapshotNonce}.html`;

    return {
      dataPoint: {
        client_slug: job.client_slug || binding.client_slug || "",
        business_slug: job.business_slug || "",
        location_slug: job.location_slug || "",
        run_id: runId,
        extension_instance_id: binding.extension_instance_id || "",
        extension_display_name: binding.extension_display_name || "",
        source_name: source.name || "",
        source_url: source.url || "",
        source_type: source.source_type || source.type || "private",
        platform: source.platform || inferPlatform(source.url),
        priority: source.priority || "",
        scan_cadence: source.scan_cadence || "",
        purpose: source.purpose || "",
        source_index: sourceIndex || 0,
        current_url: cap.url || "",
        post_url: postUrls[0] || cap.url || "",
        profile_url: profileCandidates[0] || "",
        profile_candidates: profileCandidates,
        post_candidates: postUrls,
        entity_candidates: entityItems,
        title: cap.title || "",
        visible_text_summary: excerpt.slice(0, 1200),
        raw_visible_text_excerpt: excerpt,
        engagement_hint: extractEngagementHint(text),
        captured_at: now,
        source_login_status: detectLoginHint(text),
        collector_identity: "chrome-extension-local-collector",
        confidence: text ? "medium" : "low",
        scroll_count: cap.scrollStepsUsed || 0,
        max_scrolls: cap.scrollStepsUsed || 0,
        scroll_debug: cap.scrollDebug || [],
        scroll_stopped_reason: cap.scrollStoppedReason || "",
        extraction_engine: cap.engine || "",
        read_only: true
      },
      newPrivateSources: entityItems
        .filter((it) => it && /group|community|page|channel|profile|account/i.test(String(it.type || "")))
        .slice(0, 20)
        .map((it) => ({
          run_id: runId,
          client_slug: job.client_slug || binding.client_slug || "",
          extension_instance_id: binding.extension_instance_id || "",
          extension_display_name: binding.extension_display_name || "",
          platform: source.platform || inferPlatform(source.url),
          source_type: String(it.type || "group"),
          source_name: it.name || it.url || "",
          profile_or_group_url: it.url || "unavailable",
          current_recommendation_url: cap.url || "unavailable",
          detected_while_scanning: source.name || source.url || "",
          why_relevant: "Visible platform recommendation collected for human review.",
          related_content_pillar: "",
          estimated_priority: "medium",
          suggested_scan_cadence: "weekly",
          status: "needs_human_review",
          captured_at: now,
          collector_identity: "chrome-extension-local-collector"
        })),
      snapshot: {
        run_id: runId,
        client_slug: job.client_slug || binding.client_slug || "",
        extension_instance_id: binding.extension_instance_id || "",
        extension_display_name: binding.extension_display_name || "",
        filename: snapshotFilename,
        source_name: source.name || "",
        source_url: source.url || "",
        current_url: cap.url || "",
        captured_at: now,
        html: buildSnapshotHTML({ source, job, page: pageLike, capturedAt: now })
      }
    };
  } finally {
    if (settings.closeTabsAfterCollect) {
      try {
        await chrome.tabs.remove(tab.id);
      } catch (error) {
        // Ignore tab close races.
      }
    }
  }
}

async function fetchJSON(url, options, timeoutMs) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs || 10000);
  try {
    const response = await fetch(url, { ...options, signal: controller.signal });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    return await response.json();
  } finally {
    clearTimeout(timer);
  }
}

async function postToBridge(baseUrl, token, path, payload, binding) {
  const body = { ...(payload || {}) };
  if (binding && binding.client_slug && !body.client_slug) body.client_slug = binding.client_slug;
  if (binding && binding.extension_instance_id && !body.extension_instance_id) body.extension_instance_id = binding.extension_instance_id;
  if (binding && binding.extension_display_name && !body.extension_display_name) body.extension_display_name = binding.extension_display_name;
  return fetchJSON(`${trimSlash(baseUrl)}${path}`, {
    method: "POST",
    headers: collectorHeaders(binding, {
      "Content-Type": "application/json",
      "X-Collector-Token": token
    }),
    body: JSON.stringify(body)
  }, 12000);
}

async function syncSettingsToBridge(settings) {
  return "skipped";
}

function createTab(createProperties) {
  return new Promise((resolve, reject) => {
    chrome.tabs.create(createProperties, (tab) => {
      const err = chrome.runtime.lastError;
      if (err) reject(new Error(err.message));
      else resolve(tab);
    });
  });
}

function waitForTabLoad(tabId, timeoutMs) {
  return new Promise((resolve) => {
    let done = false;
    const timer = setTimeout(() => finish(), timeoutMs || 25000);
    function finish() {
      if (done) return;
      done = true;
      clearTimeout(timer);
      chrome.tabs.onUpdated.removeListener(listener);
      resolve();
    }
    function listener(updatedTabId, changeInfo) {
      if (updatedTabId === tabId && changeInfo.status === "complete") {
        finish();
      }
    }
    chrome.tabs.onUpdated.addListener(listener);
  });
}

// Injected orchestrator for the AUTOMATED collector. Requires CAPTURE_FILES to
// already be injected (defines window.__collectorCapture). Scrolls scrollSteps
// times, accumulating the cleaned capture across scrolls, returns the final result.
async function collectCleanPage(opts) {
  const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
  const clamp = (v, lo, hi) => Math.max(lo, Math.min(hi, v));
  const maxSteps = clamp(Number(opts.maxScrollSteps) || 10, 1, 80);
  const steps = clamp(Number(opts.scrollSteps) || 5, 0, maxSteps);
  const scrollMode = String(opts.scrollMode || "standard");
  const stopAfterNoMoveScrolls = clamp(Number(opts.stopAfterNoMoveScrolls) || 0, 0, 10);
  const minD = clamp(Number(opts.minDelaySeconds) || 5, 1, 30);
  const maxD = clamp(Number(opts.maxDelaySeconds) || 5, minD, 60);
  const randomBetween = (lo, hi) => lo + Math.random() * (hi - lo);
  const delayMs = () => {
    const baseSeconds = minD + Math.random() * (maxD - minD);
    const jitter = minD === maxD ? randomBetween(0.72, 1.32) : randomBetween(0.9, 1.16);
    return Math.floor(clamp(baseSeconds * jitter, 1, 60) * 1000);
  };
  const accountUrls = [];
  const postUrls = [];
  const entityItems = [];
  const seenAccountUrls = new Set();
  const seenPostUrls = new Set();
  const seenEntityItems = new Set();
  const scrollDebug = [];

  function elementLabel(el) {
    if (!el) return "unknown";
    if (el === document.scrollingElement || el === document.documentElement || el === document.body) return "document";
    const role = el.getAttribute && el.getAttribute("role");
    const id = el.id ? `#${el.id}` : "";
    const cls = el.className && typeof el.className === "string" ? `.${el.className.trim().split(/\s+/).slice(0, 2).join(".")}` : "";
    return `${String(el.tagName || "element").toLowerCase()}${id}${cls}${role ? `[role=${role}]` : ""}`;
  }

  function scrollTopOf(target) {
    if (!target || target.kind === "document") {
      const root = document.scrollingElement || document.documentElement || document.body;
      return Math.max(window.scrollY || 0, root ? root.scrollTop || 0 : 0, document.documentElement ? document.documentElement.scrollTop || 0 : 0, document.body ? document.body.scrollTop || 0 : 0);
    }
    return Number(target.el.scrollTop || 0);
  }

  function scrollMaxOf(target) {
    if (!target || target.kind === "document") {
      const root = document.scrollingElement || document.documentElement || document.body;
      const scrollHeight = Math.max(root ? root.scrollHeight || 0 : 0, document.documentElement ? document.documentElement.scrollHeight || 0 : 0, document.body ? document.body.scrollHeight || 0 : 0);
      const clientHeight = window.innerHeight || (root ? root.clientHeight || 0 : 0);
      return Math.max(0, scrollHeight - clientHeight);
    }
    return Math.max(0, Number(target.el.scrollHeight || 0) - Number(target.el.clientHeight || 0));
  }

  function scrollCandidates() {
    const candidates = [];
    const root = document.scrollingElement || document.documentElement || document.body;
    if (root) {
      candidates.push({
        kind: "document",
        el: root,
        label: "document",
        viewport: window.innerHeight || root.clientHeight || 800,
        score: scrollMaxOf({ kind: "document", el: root }) + 1000000
      });
    }
    const selector = [
      "main",
      "section",
      "div",
      "[role='main']",
      "[role='feed']",
      "[role='list']",
      "[data-pagelet]"
    ].join(",");
    for (const el of Array.from(document.querySelectorAll(selector))) {
      if (!el || el === root || el === document.body || el === document.documentElement) continue;
      const scrollable = Number(el.scrollHeight || 0) - Number(el.clientHeight || 0);
      if (scrollable < 300) continue;
      const rect = el.getBoundingClientRect();
      if (!rect || rect.width < 220 || rect.height < 240) continue;
      const style = window.getComputedStyle ? window.getComputedStyle(el) : null;
      if (style && (style.visibility === "hidden" || style.display === "none")) continue;
      const viewportOverlap = Math.max(0, Math.min(rect.bottom, window.innerHeight || rect.bottom) - Math.max(rect.top, 0));
      const centerBonus = rect.top < (window.innerHeight || 800) * 0.75 && rect.bottom > (window.innerHeight || 800) * 0.25 ? 50000 : 0;
      candidates.push({
        kind: "element",
        el,
        label: elementLabel(el),
        viewport: Math.max(300, Math.min(rect.height, window.innerHeight || rect.height || 800)),
        score: scrollable + viewportOverlap * 10 + centerBonus
      });
    }
    return candidates
      .filter((target) => scrollMaxOf(target) - scrollTopOf(target) > 10)
      .sort((a, b) => b.score - a.score)
      .slice(0, 8);
  }

  function setScrollTopOf(target, top) {
    const nextTop = Math.max(0, Math.min(scrollMaxOf(target), Number(top) || 0));
    if (!target || target.kind === "document") {
      const root = document.scrollingElement || document.documentElement || document.body;
      if (root) root.scrollTop = nextTop;
      window.scrollTo(0, nextTop);
      return;
    }
    target.el.scrollTop = nextTop;
  }

  function easeInOutCubic(t) {
    return t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2;
  }

  function humanDurationMs(baseMs) {
    return Math.round(clamp(Number(baseMs) * randomBetween(0.72, 1.38), 420, 1450));
  }

  function humanScrollDistance(viewport, multiplier) {
    const base = Math.max(680, Math.floor((viewport || window.innerHeight || 800) * multiplier));
    return Math.round(base * randomBetween(0.84, 1.12));
  }

  async function animateScrollTarget(target, toTop, durationMs) {
    const from = scrollTopOf(target);
    const max = scrollMaxOf(target);
    const to = Math.max(0, Math.min(max, Number(toTop) || 0));
    if (Math.abs(to - from) < 1) return;
    const duration = Math.max(260, Number(durationMs) || 760);
    const raf = window.requestAnimationFrame || ((fn) => setTimeout(fn, 16));
    const start = Date.now();
    await new Promise((resolve) => {
      function frame() {
        const elapsed = Date.now() - start;
        const progress = Math.min(1, elapsed / duration);
        const nextTop = from + (to - from) * easeInOutCubic(progress);
        setScrollTopOf(target, nextTop);
        if (progress < 1) {
          raf(frame);
        } else {
          resolve();
        }
      }
      raf(frame);
    });
    setScrollTopOf(target, to);
  }

  async function tryScrollTarget(target, distance) {
    const before = scrollTopOf(target);
    const nextTop = Math.min(scrollMaxOf(target), before + distance);
    const durationMs = humanDurationMs(scrollMode === "discovery" ? 850 : 700);
    await animateScrollTarget(target, nextTop, durationMs);
    await wait(Math.round(randomBetween(60, 170)));
    const after = scrollTopOf(target);
    return {
      target: target.label,
      before: Math.round(before),
      after: Math.round(after),
      delta: Math.round(after - before),
      requested_distance: Math.round(distance),
      duration_ms: durationMs,
      max: Math.round(scrollMaxOf(target))
    };
  }

  async function performCollectorScroll() {
    const multiplier = scrollMode === "discovery" ? 0.95 : 0.85;
    const targets = scrollCandidates();
    let best = null;
    for (const target of targets) {
      const distance = humanScrollDistance(target.viewport || window.innerHeight || 800, multiplier);
      const report = await tryScrollTarget(target, distance);
      if (!best || Math.abs(report.delta) > Math.abs(best.delta || 0)) best = report;
      if (Math.abs(report.delta) >= 120) return report;
    }
    if (best) return best;
    const fallbackDistance = humanScrollDistance(window.innerHeight || 800, multiplier);
    const fallbackTarget = { kind: "document", el: document.scrollingElement || document.documentElement || document.body, label: "window-fallback" };
    const durationMs = humanDurationMs(scrollMode === "discovery" ? 850 : 700);
    await animateScrollTarget(fallbackTarget, scrollTopOf(fallbackTarget) + fallbackDistance, durationMs);
    await wait(Math.round(randomBetween(60, 170)));
    return {
      target: "window-fallback",
      before: null,
      after: Math.round(window.scrollY || 0),
      delta: null,
      requested_distance: Math.round(fallbackDistance),
      duration_ms: durationMs,
      max: Math.round(scrollMaxOf({ kind: "document", el: document.scrollingElement || document.documentElement || document.body }))
    };
  }

  function urlKey(value) {
    try {
      const parsed = new URL(String(value || ""), location.href);
      parsed.hash = "";
      return parsed.href.replace(/\/+$/, "").toLowerCase();
    } catch (error) {
      return String(value || "").trim().replace(/\/+$/, "").toLowerCase();
    }
  }

  function addUniqueUrl(target, seen, values) {
    if (!Array.isArray(values)) return;
    for (const value of values) {
      if (!value) continue;
      const key = urlKey(value);
      if (!key || seen.has(key)) continue;
      seen.add(key);
      target.push(value);
    }
  }

  function entityKey(item) {
    if (!item || typeof item !== "object") return "";
    if (item.url) return urlKey(item.url);
    return `${String(item.type || "").toLowerCase()}:${String(item.name || "").trim().toLowerCase()}`;
  }

  function addEntityItems(values) {
    if (!Array.isArray(values)) return;
    for (const item of values) {
      const key = entityKey(item);
      if (!key || seenEntityItems.has(key)) continue;
      seenEntityItems.add(key);
      entityItems.push(item);
    }
  }

  let prev = "";
  let last = null;
  let scrollStepsUsed = 0;
  let consecutiveNoMove = 0;
  let stoppedReason = "";
  for (let i = 0; i <= steps; i += 1) {
    try {
      last = (typeof window.__collectorCapture === "function") ? window.__collectorCapture(prev, {}) : last;
    } catch (error) {
      last = { error: String(error && error.message ? error.message : error) };
    }
    if (last) {
      addUniqueUrl(accountUrls, seenAccountUrls, last.accountUrls);
      addUniqueUrl(postUrls, seenPostUrls, last.postUrls);
      addEntityItems(last.entityItems);
    }
    if (last && last.merged) prev = last.merged;
    if (i < steps) {
      const scrollReport = await performCollectorScroll();
      scrollStepsUsed += 1;
      scrollReport.pause_ms = delayMs();
      scrollDebug.push(scrollReport);
      if (!scrollReport || Math.abs(Number(scrollReport.delta || 0)) < 80) {
        consecutiveNoMove += 1;
      } else {
        consecutiveNoMove = 0;
      }
      if (stopAfterNoMoveScrolls > 0 && consecutiveNoMove >= stopAfterNoMoveScrolls) {
        stoppedReason = `no_scroll_movement_${consecutiveNoMove}_times`;
        break;
      }
      await wait(scrollReport.pause_ms);
    }
  }
  if (last) {
    last.scrollStepsUsed = scrollStepsUsed;
    last.requestedScrollSteps = steps;
    last.scrollStoppedReason = stoppedReason;
    last.scrollDebug = scrollDebug.slice(-20);
    last.accountUrls = accountUrls;
    last.postUrls = postUrls;
    last.entityItems = entityItems;
  }
  return last || {};
}

function extractEngagementHint(text) {
  const m = String(text || "").match(/[0-9][0-9.,KM]*\s*(reactions?|likes?|comments?|shares?|views?|replies|reply|reposts?|retweets?|answers?)/gi);
  return m ? Array.from(new Set(m)).slice(0, 12).join("; ") : "";
}

function detectLoginHint(text) {
  const lower = String(text || "").toLowerCase();
  if (lower.includes("log in") || lower.includes("sign in") || lower.includes("login")) return "maybe_logged_out";
  return text ? "available" : "unknown";
}

function collectVisiblePage(source, job, options) {
  const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
  const clamp = (value, min, max) => Math.max(min, Math.min(max, value));
  const minDelay = clamp(Number(options.minDelaySeconds || 5), 5, 20);
  const maxDelay = clamp(Number(options.maxDelaySeconds || 5), minDelay, 30);
  const delayMs = () => Math.floor((minDelay + Math.random() * (maxDelay - minDelay)) * 1000);
  const maxTextChars = clamp(Number(options.maxTextChars || 12000), 1000, 30000);
  const scrollSteps = clamp(Number(options.scrollSteps || 5), 0, 10);

  function cleanText(text) {
    return String(text || "").replace(/\s+/g, " ").trim();
  }

  function textFromVisibleBody() {
    const body = document.body;
    if (!body) return "";
    return cleanText(body.innerText || body.textContent || "");
  }

  function collectLinks() {
    const anchors = Array.from(document.querySelectorAll("a[href]"));
    return anchors.slice(0, 300).map((a) => {
      let href = a.href || "";
      try {
        const parsed = new URL(href, location.href);
        parsed.hash = "";
        href = parsed.toString();
      } catch (error) {
        // Keep original href.
      }
      return {
        text: cleanText(a.innerText || a.getAttribute("aria-label") || a.title || "").slice(0, 160),
        href
      };
    }).filter((item) => item.href && /^https?:\/\//i.test(item.href));
  }

  function inferProfileCandidates(links) {
    const platform = String(source.platform || "").toLowerCase();
    const sourceURL = String(source.url || "");
    const candidates = [];
    if (sourceURL) candidates.push(sourceURL);
    for (const link of links) {
      const href = link.href || "";
      if (platform === "facebook" && /facebook\.com\/(profile\.php\?id=|people\/|pages\/|[A-Za-z0-9_.-]+\/?$)/.test(href)) candidates.push(href);
      if (platform === "linkedin" && /linkedin\.com\/(in|company)\//.test(href)) candidates.push(href);
      if (platform === "instagram" && /instagram\.com\/[A-Za-z0-9_.]+\/?$/.test(href)) candidates.push(href);
      if (platform === "tiktok" && /tiktok\.com\/@/.test(href)) candidates.push(href);
      if (platform === "reddit" && /reddit\.com\/(user|r)\//.test(href)) candidates.push(href);
      if ((platform === "x" || platform === "twitter") && /(x|twitter)\.com\/[A-Za-z0-9_]+\/?$/.test(href)) candidates.push(href);
    }
    return Array.from(new Set(candidates)).slice(0, 20);
  }

  function inferPostCandidates(links) {
    const candidates = [location.href];
    for (const link of links) {
      const href = link.href || "";
      if (/\/posts\/|\/permalink\/|\/videos\/|\/reel\/|\/status\/|\/comments\/|\/comment\//i.test(href)) candidates.push(href);
      if (/reddit\.com\/r\/[^/]+\/comments\//i.test(href)) candidates.push(href);
      if (/linkedin\.com\/feed\/update\//i.test(href)) candidates.push(href);
    }
    return Array.from(new Set(candidates)).slice(0, 20);
  }

  function inferNewPrivateSourceCandidates(links) {
    const candidates = [];
    for (const link of links) {
      const href = link.href || "";
      const text = link.text || "";
      if (!href) continue;
      const isLikelyGroup =
        /facebook\.com\/groups\//i.test(href) ||
        /linkedin\.com\/groups\//i.test(href) ||
        /reddit\.com\/r\//i.test(href);
      const isLikelyPage =
        /facebook\.com\/pages\//i.test(href) ||
        /linkedin\.com\/company\//i.test(href);
      const hasRecommendationText = /suggest|recommend|related|similar|groups?|communities|pages?/i.test(text);
      if ((isLikelyGroup || isLikelyPage) && (hasRecommendationText || text.length > 2)) {
        candidates.push({
          text: text || href,
          href,
          source_type: isLikelyGroup ? "group" : "page"
        });
      }
    }
    return Array.from(new Map(candidates.map((item) => [item.href, item])).values()).slice(0, 20);
  }

  function engagementHint(text) {
    const matches = text.match(/\b([0-9][0-9,.]*\s*(comments?|likes?|shares?|views?|reactions?|replies?|saves?))\b/gi);
    return matches ? Array.from(new Set(matches)).slice(0, 10).join("; ") : "";
  }

  function loginHint(text) {
    const lower = text.toLowerCase();
    if (lower.includes("log in") || lower.includes("sign in") || lower.includes("login")) {
      return "maybe_logged_out";
    }
    return "available";
  }

  return (async () => {
    for (let i = 0; i < scrollSteps; i += 1) {
      window.scrollBy({ top: Math.max(300, Math.floor(window.innerHeight * 0.75)), left: 0, behavior: "smooth" });
      await wait(delayMs());
    }
    const rawText = textFromVisibleBody();
    const links = collectLinks();
    const profileCandidates = inferProfileCandidates(links);
    const postCandidates = inferPostCandidates(links);
    const newPrivateSourceCandidates = inferNewPrivateSourceCandidates(links);
    const excerpt = rawText.slice(0, maxTextChars);
    return {
      title: document.title || "",
      current_url: location.href,
      profile_url: profileCandidates[0] || "",
      post_url: postCandidates[0] || location.href,
      profile_candidates: profileCandidates,
      post_candidates: postCandidates,
      visible_text_summary: excerpt.slice(0, 1200),
      raw_visible_text_excerpt: excerpt,
      engagement_hint: engagementHint(rawText),
      login_hint: loginHint(rawText),
      link_count: links.length,
      scroll_count: scrollSteps,
      max_scrolls: scrollSteps,
      new_private_source_candidates: newPrivateSourceCandidates,
      sample_links: links.slice(0, 80)
    };
  })();
}

function buildSnapshotHTML({ source, job, page, capturedAt }) {
  const safe = escapeHTML;
  const links = [
    ...(page.profile_candidates || []).map((url) => ({ label: "Profile candidate", url })),
    ...(page.post_candidates || []).map((url) => ({ label: "Post/current candidate", url }))
  ].slice(0, 40);

  return `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${safe(source.name || page.title || "Collector snapshot")}</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 24px; line-height: 1.45; color: #17202a; }
    main { max-width: 920px; margin: 0 auto; }
    section { border: 1px solid #d8dee4; border-radius: 8px; padding: 16px; margin: 16px 0; }
    h1 { font-size: 22px; }
    h2 { font-size: 16px; margin-top: 0; }
    pre { white-space: pre-wrap; word-break: break-word; background: #f6f8fa; padding: 12px; border-radius: 6px; }
    a { color: #0969da; word-break: break-word; }
    .meta { color: #57606a; font-size: 13px; }
  </style>
</head>
<body>
  <main>
    <h1>${safe(source.name || "Collector snapshot")}</h1>
    <p class="meta">Captured at ${safe(capturedAt)} by Chrome Extension Local Collector. This is a local verification snapshot, not a cloud upload.</p>
    <section>
      <h2>Source</h2>
      <p><strong>Client:</strong> ${safe(job.client_slug || "")}</p>
      <p><strong>Platform:</strong> ${safe(source.platform || "")}</p>
      <p><strong>Configured URL:</strong> <a href="${safe(source.url || "")}">${safe(source.url || "")}</a></p>
      <p><strong>Current URL:</strong> <a href="${safe(page.current_url || "")}">${safe(page.current_url || "")}</a></p>
    </section>
    <section>
      <h2>Candidate URLs</h2>
      <ul>${links.map((link) => `<li>${safe(link.label)}: <a href="${safe(link.url)}">${safe(link.url)}</a></li>`).join("")}</ul>
    </section>
    <section>
      <h2>Visible Text Excerpt</h2>
      <pre>${safe(page.raw_visible_text_excerpt || "")}</pre>
    </section>
  </main>
</body>
</html>`;
}

function normalizeSources(sources) {
  if (!Array.isArray(sources)) return [];
  const seen = new Set();
  const normalized = [];
  for (const source of sources) {
    if (!source || typeof source !== "object" || !source.url) continue;
    const sourceUrl = String(source.url || "").trim();
    const key = canonicalSourceKey(sourceUrl);
    if (!key || seen.has(key)) continue;
    seen.add(key);
    normalized.push({
      ...source,
      url: sourceUrl,
      platform: source.platform || inferPlatform(sourceUrl)
    });
  }
  return normalized;
}

function normalizeSettings(input) {
  const next = { ...DEFAULT_SETTINGS, ...input };
  next.enabled = Boolean(next.enabled);
  next.bridgeBaseUrl = trimSlash(String(next.bridgeBaseUrl || DEFAULT_SETTINGS.bridgeBaseUrl));
  next.pollMinutes = clampNumber(next.pollMinutes, 1, 60, 1);
  next.pollSeconds = clampNumber(next.pollSeconds, 5, 60, 5);
  next.minDelaySeconds = clampNumber(next.minDelaySeconds, 5, 20, 5);
  next.maxDelaySeconds = clampNumber(next.maxDelaySeconds, next.minDelaySeconds, 30, 5);
  next.maxSourcesPerRun = clampNumber(next.maxSourcesPerRun, 1, 50, 20);
  next.sourceConcurrency = clampNumber(next.sourceConcurrency, 1, 3, 1);
  next.scrollSteps = clampNumber(next.scrollSteps, 0, 10, 5);
  next.maxTextChars = clampNumber(next.maxTextChars, 1000, 30000, 12000);
  next.closeTabsAfterCollect = Boolean(next.closeTabsAfterCollect);
  return next;
}

function normalizeClientBinding(input) {
  const raw = input && typeof input === "object" ? input : {};
  return {
    client_slug: String(raw.client_slug || "").trim(),
    client_name: String(raw.client_name || "").trim(),
    extension_instance_id: String(raw.extension_instance_id || "").trim(),
    extension_display_name: String(raw.extension_display_name || raw.client_name || "").trim(),
    bridge_base_url: raw.bridge_base_url ? trimSlash(String(raw.bridge_base_url)) : ""
  };
}

async function getClientBinding() {
  if (clientBindingCache) return clientBindingCache;
  try {
    const response = await fetch(chrome.runtime.getURL("client_binding.json"), { cache: "no-store" });
    if (response.ok) {
      clientBindingCache = normalizeClientBinding(await response.json());
      return clientBindingCache;
    }
  } catch (error) {
    // Legacy single-client installs may not have a binding file yet.
  }
  clientBindingCache = normalizeClientBinding({});
  return clientBindingCache;
}

function collectorHeaders(binding, extra) {
  const headers = {
    "X-Collector-Extension": "media-agency-local-collector",
    "X-Collector-Extension-Version": EXTENSION_BUILD,
    ...(extra || {})
  };
  if (binding && binding.client_slug) headers["X-Collector-Client-Slug"] = binding.client_slug;
  if (binding && binding.extension_instance_id) headers["X-Collector-Extension-Instance"] = binding.extension_instance_id;
  if (binding && binding.extension_display_name) headers["X-Collector-Extension-Name"] = binding.extension_display_name;
  return headers;
}

function jobMatchesBinding(job, binding) {
  if (!job || !binding) return true;
  const jobClient = String(job.client_slug || "").trim();
  if (jobClient && binding.client_slug && jobClient !== binding.client_slug) return false;
  if (jobClient && !binding.client_slug) return false;
  const jobExtension = String(job.extension_instance_id || "").trim();
  if (jobExtension && binding.extension_instance_id && jobExtension !== binding.extension_instance_id) return false;
  if (jobExtension && !binding.extension_instance_id) return false;
  const allowed = Array.isArray(job.allowed_extension_instance_ids)
    ? job.allowed_extension_instance_ids.map((item) => String(item || "").trim()).filter(Boolean)
    : [];
  if (allowed.length > 0) {
    return Boolean(binding.extension_instance_id && allowed.includes(binding.extension_instance_id));
  }
  return true;
}

async function getSettings() {
  const data = await chrome.storage.local.get(SETTINGS_KEY);
  const binding = await getClientBinding();
  const settings = normalizeSettings(data[SETTINGS_KEY] || DEFAULT_SETTINGS);
  if (binding.bridge_base_url && (!data[SETTINGS_KEY] || !data[SETTINGS_KEY].bridgeBaseUrl)) {
    settings.bridgeBaseUrl = binding.bridge_base_url;
  }
  return settings;
}

async function getState() {
  const data = await chrome.storage.local.get(STATE_KEY);
  return data[STATE_KEY] || { status: "idle", message: "Collector idle." };
}

async function getAudit() {
  const data = await chrome.storage.local.get(AUDIT_KEY);
  return data[AUDIT_KEY] || null;
}

// Manual single-scroll capture for the audit panel. The operator opens a page,
// clicks Capture, scrolls, clicks Capture again -- each call injects the
// capture modules into the active tab, serializes the current DOM (feed vs
// website routed by platform), and merges with the text accumulated so far for
// the same URL. This mirrors what the automated collector does per scroll step.
async function manualCapture() {
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  const tab = tabs && tabs[0];
  if (!tab || !tab.id) return { ok: false, error: "No active tab." };

  const tabUrl = tab.url || "";
  if (!/^https?:\/\//i.test(tabUrl)) {
    return { ok: false, error: "Active tab is not an http(s) page." };
  }

  try {
    await chrome.scripting.executeScript({ target: { tabId: tab.id }, files: CAPTURE_FILES });
  } catch (error) {
    return { ok: false, error: "Inject failed: " + (error && error.message ? error.message : error) };
  }

  const audit = await getAudit();
  const samePage = audit && audit.url === tabUrl;
  const prevText = samePage ? (audit.text || "") : "";

  let out;
  try {
    const [res] = await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: (prev) => window.__collectorCapture(prev, {}),
      args: [prevText]
    });
    out = res && res.result ? res.result : null;
  } catch (error) {
    return { ok: false, error: "Capture failed: " + (error && error.message ? error.message : error) };
  }

  if (!out) return { ok: false, error: "No capture result." };
  if (out.error) return { ok: false, error: "Capture error: " + out.error };

  const prevCount = samePage ? (audit.count || 0) : 0;
  const prevCaptures = samePage ? (audit.captures || []) : [];
  const nextAudit = {
    url: tabUrl,
    title: out.title || tab.title || "",
    platform: out.platform,
    branch: out.branch,
    engine: out.engine || "",
    count: prevCount + 1,
    text: out.merged,                 // absolute-URL accumulator (next prevText)
    displayText: out.mergedDisplay || out.merged, // shortened, LLM-facing / audit view
    lastCaptureText: out.captureText,
    captures: prevCaptures.concat([{
      n: prevCount + 1,
      engine: out.engine || "",
      scrollY: out.scrollY,
      captureChars: out.captureChars,
      captureUrls: out.captureUrls,
      mergedChars: out.mergedChars,
      mergedUrls: out.mergedUrls,
      droppedUnits: out.droppedUnits || 0,
      keptUnits: out.keptUnits || 0,
      capturedAt: new Date().toISOString()
    }]),
    updatedAt: new Date().toISOString()
  };
  await chrome.storage.local.set({ [AUDIT_KEY]: nextAudit });
  return { ok: true, audit: nextAudit };
}

async function testScrollActiveTab(message) {
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  const tab = tabs && tabs[0];
  if (!tab || !tab.id) return { ok: false, error: "No active tab." };
  const tabUrl = tab.url || "";
  if (!/^https?:\/\//i.test(tabUrl)) {
    return { ok: false, error: "Active tab is not an http(s) page." };
  }
  const steps = clampNumber(message.steps, 1, 20, 8);
  const delayMs = clampNumber(message.delay_ms, 250, 5000, 900);
  try {
    const [result] = await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: runVisibleScrollTest,
      args: [{ steps, delayMs }]
    });
    const payload = result && result.result ? result.result : {};
    return {
      ok: true,
      url: tabUrl,
      steps_requested: steps,
      steps_used: payload.stepsUsed || 0,
      debug: payload.debug || []
    };
  } catch (error) {
    return { ok: false, error: String(error && error.message ? error.message : error) };
  }
}

async function runVisibleScrollTest(opts) {
  const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
  const clamp = (v, lo, hi) => Math.max(lo, Math.min(hi, Number(v) || 0));
  const steps = clamp(opts && opts.steps, 1, 20);
  const delayMs = clamp(opts && opts.delayMs, 250, 5000);
  const randomBetween = (lo, hi) => lo + Math.random() * (hi - lo);
  const debug = [];

  function elementLabel(el) {
    if (!el) return "unknown";
    if (el === document.scrollingElement || el === document.documentElement || el === document.body) return "document";
    const role = el.getAttribute && el.getAttribute("role");
    const id = el.id ? `#${el.id}` : "";
    const cls = el.className && typeof el.className === "string" ? `.${el.className.trim().split(/\s+/).slice(0, 2).join(".")}` : "";
    return `${String(el.tagName || "element").toLowerCase()}${id}${cls}${role ? `[role=${role}]` : ""}`;
  }

  function scrollTopOf(target) {
    if (!target || target.kind === "document") {
      const root = document.scrollingElement || document.documentElement || document.body;
      return Math.max(window.scrollY || 0, root ? root.scrollTop || 0 : 0, document.documentElement ? document.documentElement.scrollTop || 0 : 0, document.body ? document.body.scrollTop || 0 : 0);
    }
    return Number(target.el.scrollTop || 0);
  }

  function scrollMaxOf(target) {
    if (!target || target.kind === "document") {
      const root = document.scrollingElement || document.documentElement || document.body;
      const scrollHeight = Math.max(root ? root.scrollHeight || 0 : 0, document.documentElement ? document.documentElement.scrollHeight || 0 : 0, document.body ? document.body.scrollHeight || 0 : 0);
      const clientHeight = window.innerHeight || (root ? root.clientHeight || 0 : 0);
      return Math.max(0, scrollHeight - clientHeight);
    }
    return Math.max(0, Number(target.el.scrollHeight || 0) - Number(target.el.clientHeight || 0));
  }

  function scrollCandidates() {
    const candidates = [];
    const root = document.scrollingElement || document.documentElement || document.body;
    if (root) {
      candidates.push({
        kind: "document",
        el: root,
        label: "document",
        viewport: window.innerHeight || root.clientHeight || 800,
        score: scrollMaxOf({ kind: "document", el: root }) + 1000000
      });
    }
    const selector = [
      "main",
      "section",
      "div",
      "[role='main']",
      "[role='feed']",
      "[role='list']",
      "[data-pagelet]"
    ].join(",");
    for (const el of Array.from(document.querySelectorAll(selector))) {
      if (!el || el === root || el === document.body || el === document.documentElement) continue;
      const scrollable = Number(el.scrollHeight || 0) - Number(el.clientHeight || 0);
      if (scrollable < 300) continue;
      const rect = el.getBoundingClientRect();
      if (!rect || rect.width < 220 || rect.height < 240) continue;
      const style = window.getComputedStyle ? window.getComputedStyle(el) : null;
      if (style && (style.visibility === "hidden" || style.display === "none")) continue;
      const viewportOverlap = Math.max(0, Math.min(rect.bottom, window.innerHeight || rect.bottom) - Math.max(rect.top, 0));
      const centerBonus = rect.top < (window.innerHeight || 800) * 0.75 && rect.bottom > (window.innerHeight || 800) * 0.25 ? 50000 : 0;
      candidates.push({
        kind: "element",
        el,
        label: elementLabel(el),
        viewport: Math.max(300, Math.min(rect.height, window.innerHeight || rect.height || 800)),
        score: scrollable + viewportOverlap * 10 + centerBonus
      });
    }
    return candidates
      .filter((target) => scrollMaxOf(target) - scrollTopOf(target) > 10)
      .sort((a, b) => b.score - a.score)
      .slice(0, 8);
  }

  function setScrollTopOf(target, top) {
    const nextTop = Math.max(0, Math.min(scrollMaxOf(target), Number(top) || 0));
    if (!target || target.kind === "document") {
      const root = document.scrollingElement || document.documentElement || document.body;
      if (root) root.scrollTop = nextTop;
      window.scrollTo(0, nextTop);
      return;
    }
    target.el.scrollTop = nextTop;
  }

  function easeInOutCubic(t) {
    return t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2;
  }

  function humanDurationMs(baseMs) {
    return Math.round(clamp(Number(baseMs) * randomBetween(0.72, 1.38), 420, 1450));
  }

  function humanPauseMs(baseMs) {
    return Math.round(clamp(Number(baseMs) * randomBetween(0.72, 1.34), 250, 5000));
  }

  function humanScrollDistance(viewport) {
    const base = Math.max(680, Math.floor((viewport || window.innerHeight || 800) * 0.95));
    return Math.round(base * randomBetween(0.84, 1.12));
  }

  async function animateScrollTarget(target, toTop, durationMs) {
    const from = scrollTopOf(target);
    const max = scrollMaxOf(target);
    const to = Math.max(0, Math.min(max, Number(toTop) || 0));
    if (Math.abs(to - from) < 1) return;
    const duration = Math.max(260, Number(durationMs) || 850);
    const raf = window.requestAnimationFrame || ((fn) => setTimeout(fn, 16));
    const start = Date.now();
    await new Promise((resolve) => {
      function frame() {
        const elapsed = Date.now() - start;
        const progress = Math.min(1, elapsed / duration);
        const nextTop = from + (to - from) * easeInOutCubic(progress);
        setScrollTopOf(target, nextTop);
        if (progress < 1) {
          raf(frame);
        } else {
          resolve();
        }
      }
      raf(frame);
    });
    setScrollTopOf(target, to);
  }

  async function tryScrollTarget(target, distance) {
    const before = scrollTopOf(target);
    const nextTop = Math.min(scrollMaxOf(target), before + distance);
    const durationMs = humanDurationMs(850);
    await animateScrollTarget(target, nextTop, durationMs);
    await wait(Math.round(randomBetween(60, 170)));
    const after = scrollTopOf(target);
    return {
      target: target.label,
      before: Math.round(before),
      after: Math.round(after),
      delta: Math.round(after - before),
      requested_distance: Math.round(distance),
      duration_ms: durationMs,
      max: Math.round(scrollMaxOf(target))
    };
  }

  async function performScroll() {
    const targets = scrollCandidates();
    let best = null;
    for (const target of targets) {
      const distance = humanScrollDistance(target.viewport || window.innerHeight || 800);
      const report = await tryScrollTarget(target, distance);
      if (!best || Math.abs(report.delta) > Math.abs(best.delta || 0)) best = report;
      if (Math.abs(report.delta) >= 120) return report;
    }
    return best || { target: "none", before: 0, after: 0, delta: 0, max: 0 };
  }

  for (let i = 0; i < steps; i += 1) {
    const report = await performScroll();
    report.pause_ms = humanPauseMs(delayMs);
    debug.push(report);
    await wait(report.pause_ms);
  }
  return { stepsUsed: debug.length, debug };
}

async function setState(patch) {
  const current = await getState();
  await chrome.storage.local.set({
    [STATE_KEY]: {
      ...current,
      ...patch,
      updatedAt: patch.updatedAt || new Date().toISOString()
    }
  });
}

async function getCompletedRuns() {
  const data = await chrome.storage.local.get(COMPLETED_RUNS_KEY);
  return data[COMPLETED_RUNS_KEY] || {};
}

async function acquireRunLock(runId, job, bridgeStatus, reason) {
  const now = Date.now();
  if (inMemoryActiveRuns.size > 0) {
    return { acquired: false, reason: "in_memory_active_run" };
  }

  const lockData = await chrome.storage.local.get(ACTIVE_RUN_KEY);
  if (inMemoryActiveRuns.size > 0) {
    return { acquired: false, reason: "in_memory_active_run" };
  }

  const currentLock = lockData[ACTIVE_RUN_KEY] || null;
  const lockExpiresAt = currentLock && Date.parse(currentLock.expiresAt || "");
  if (currentLock && Number.isFinite(lockExpiresAt) && lockExpiresAt > now) {
    return { acquired: false, reason: "storage_active_run", lock: currentLock };
  }

  const owner = `${runId}-${now}-${Math.random().toString(36).slice(2)}`;
  const lock = {
    runId,
    owner,
    reason,
    bridgeRunId: String(bridgeStatus?.run_id || ""),
    acquiredAt: new Date(now).toISOString(),
    expiresAt: new Date(now + activeRunLockMs(job)).toISOString()
  };

  inMemoryActiveRuns.add(runId);
  await chrome.storage.local.set({ [ACTIVE_RUN_KEY]: lock });

  const verifyData = await chrome.storage.local.get(ACTIVE_RUN_KEY);
  const verified = verifyData[ACTIVE_RUN_KEY] || {};
  if (verified.owner !== owner) {
    inMemoryActiveRuns.delete(runId);
    return { acquired: false, reason: "storage_lock_lost", lock: verified };
  }

  return { acquired: true, owner, lock };
}

async function releaseRunLock(runId, owner) {
  inMemoryActiveRuns.delete(runId);
  const lockData = await chrome.storage.local.get(ACTIVE_RUN_KEY);
  const currentLock = lockData[ACTIVE_RUN_KEY] || null;
  if (currentLock && currentLock.runId === runId && currentLock.owner === owner) {
    await chrome.storage.local.remove(ACTIVE_RUN_KEY);
  }
}

function trimCompletedRuns(runs) {
  const entries = Object.entries(runs).slice(-100);
  return Object.fromEntries(entries);
}

function activeRunLockMs(job) {
  const pacing = job?.pacing || {};
  const sources = normalizeSources(job?.sources || []);
  const maxSources = Math.min(
    Number(pacing.max_sources || DEFAULT_SETTINGS.maxSourcesPerRun || 20),
    sources.length || Number(pacing.max_sources || DEFAULT_SETTINGS.maxSourcesPerRun || 20)
  );
  const scrollCap = isDiscoveryCollection(job) ? DISCOVERY_SCROLL_CAP : NORMAL_SCROLL_CAP;
  const scrollSteps = clampNumber(pacing.scroll_steps, 0, scrollCap, DEFAULT_SETTINGS.scrollSteps);
  const maxDelaySeconds = clampNumber(pacing.max_delay_seconds, 5, 30, DEFAULT_SETTINGS.maxDelaySeconds);
  const estimatedMs = maxSources * ((scrollSteps + 2) * maxDelaySeconds * 1000 + 45000);
  const fallbackMs = ACTIVE_RUN_LOCK_MINUTES * 60 * 1000;
  return Math.max(30 * 60 * 1000, Math.min(fallbackMs, estimatedMs || fallbackMs));
}

function textLooksLikeDiscovery(value) {
  return /\b(discover|discovery|joined|joins|member|membership|following|subscriptions?|recommendation|source[_ -]?discovery|group[_ -]?discovery)\b/i.test(String(value || ""));
}

function urlLooksLikeDiscovery(url) {
  const value = String(url || "").toLowerCase();
  return (
    /facebook\.com\/groups\/joins\b/.test(value) ||
    /facebook\.com\/groups\/discover\b/.test(value) ||
    /reddit\.com\/subreddits\/mine\b/.test(value) ||
    /linkedin\.com\/mynetwork\b/.test(value) ||
    /youtube\.com\/feed\/subscriptions\b/.test(value)
  );
}

function sourceLooksLikeDiscovery(source) {
  if (!source || typeof source !== "object") return false;
  return (
    urlLooksLikeDiscovery(source.url) ||
    textLooksLikeDiscovery(source.name) ||
    textLooksLikeDiscovery(source.source_type) ||
    textLooksLikeDiscovery(source.type) ||
    textLooksLikeDiscovery(source.purpose) ||
    textLooksLikeDiscovery(source.scan_mode) ||
    textLooksLikeDiscovery(source.collection_mode)
  );
}

function isDiscoveryCollection(job, source) {
  if (sourceLooksLikeDiscovery(source)) return true;
  if (!job || typeof job !== "object") return false;
  if (
    textLooksLikeDiscovery(job.job_type) ||
    textLooksLikeDiscovery(job.mode) ||
    textLooksLikeDiscovery(job.purpose) ||
    textLooksLikeDiscovery(job.collection_mode) ||
    textLooksLikeDiscovery(job.scan_mode)
  ) {
    return true;
  }
  return normalizeSources(job.sources || []).some(sourceLooksLikeDiscovery);
}

function maxScrollStepsForCollection(job, source) {
  return isDiscoveryCollection(job, source) ? DISCOVERY_SCROLL_CAP : NORMAL_SCROLL_CAP;
}

function canonicalSourceKey(url) {
  try {
    const parsed = new URL(String(url || "").trim());
    parsed.hash = "";
    for (const param of [
      "__cft__",
      "__tn__",
      "mibextid",
      "ref",
      "refsrc",
      "tracking",
      "utm_campaign",
      "utm_content",
      "utm_medium",
      "utm_source",
      "utm_term"
    ]) {
      parsed.searchParams.delete(param);
    }
    return parsed.toString().replace(/\/+$/, "").toLowerCase();
  } catch (error) {
    return String(url || "").trim().replace(/\/+$/, "").toLowerCase();
  }
}

function trimSlash(value) {
  return String(value || "").replace(/\/+$/, "");
}

function randomDelayMs(settings, pacing) {
  const min = Number(pacing.min_delay_seconds || settings.minDelaySeconds || 5);
  const max = Math.max(min, Number(pacing.max_delay_seconds || settings.maxDelaySeconds || 5));
  return Math.floor((min + Math.random() * (max - min)) * 1000);
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function clampNumber(value, min, max, fallback) {
  const n = Number(value);
  if (!Number.isFinite(n)) return fallback;
  return Math.max(min, Math.min(max, n));
}

function inferPlatform(url) {
  const lower = String(url || "").toLowerCase();
  if (lower.includes("facebook.com")) return "facebook";
  if (lower.includes("linkedin.com")) return "linkedin";
  if (lower.includes("instagram.com")) return "instagram";
  if (lower.includes("tiktok.com")) return "tiktok";
  if (lower.includes("reddit.com")) return "reddit";
  if (lower.includes("youtube.com") || lower.includes("youtu.be")) return "youtube";
  if (lower.includes("x.com") || lower.includes("twitter.com")) return "x";
  return "web";
}

function isCompetitorSource(source) {
  const purpose = String(source.purpose || source.category || source.kind || "").toLowerCase();
  return purpose.includes("competitor") || source.is_competitor === true;
}

function safeSlug(value) {
  return String(value || "item").toLowerCase().replace(/[^a-z0-9._-]+/g, "_").replace(/^_+|_+$/g, "").slice(0, 80) || "item";
}

function escapeHTML(value) {
  return String(value || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}
