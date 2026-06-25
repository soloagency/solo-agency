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
const BUILD_STATE_KEY = "collector_extension_build";
const CAPTURE_FILES = ["collector_helpers.js", "readability.js", "filtering.js", "infinity_loops.js"];
const ACTIVE_RUN_LOCK_MINUTES = 120;
const EXTENSION_BUILD = "0.1.16-no-window-focus";
const NORMAL_SCROLL_CAP = 10;
const DISCOVERY_SCROLL_CAP = 10;

const inMemoryActiveRuns = new Set();
let clientBindingCache = null;

chrome.runtime.onInstalled.addListener(async () => {
  await resetRunLockAfterBuildChange("installed");
  const settings = await getSettings();
  await chrome.storage.local.set({ [SETTINGS_KEY]: settings });
  chrome.alarms.create("collector_poll", { periodInMinutes: Math.max(1, settings.pollMinutes || 1) });
  scheduleShortPoll(settings);
  await setState({ status: "installed", message: "Collector installed and idle." });
  await pollBridge("installed");
});

chrome.runtime.onStartup.addListener(async () => {
  await resetRunLockAfterBuildChange("startup");
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
  await resetRunLockAfterBuildChange(reason || "poll");
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
    const tabActivationPlan = collectionTabActivationPlan(job, source);
    const tabActivationMode = tabActivationPlan.mode;
    await postToBridge(bridgeBaseUrl, token, "/collect/source_status", {
      run_id: runId,
      client_slug: job.client_slug || binding.client_slug || "",
      source_name: source.name || "",
      source_url: source.url || "",
      platform: source.platform || "",
      status: "started",
      tab_activation_mode: tabActivationMode,
      window_focus_requested: tabActivationPlan.focusWindow,
      tab_create_active: tabActivationPlan.createActive,
      tab_update_active: tabActivationPlan.updateActive,
      window_focus_policy: tabActivationPlan.focusWindow ? "focus_window" : "keep_window_background",
      capture_overlay: tabActivationMode !== "background_tab",
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
        tab_activation_mode: tabActivationMode,
        window_focus_requested: tabActivationPlan.focusWindow,
        tab_create_active: tabActivationPlan.createActive,
        tab_update_active: tabActivationPlan.updateActive,
        window_focus_policy: tabActivationPlan.focusWindow ? "focus_window" : "keep_window_background",
        capture_overlay: tabActivationMode !== "background_tab",
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
        tab_activation_mode: tabActivationMode,
        window_focus_requested: tabActivationPlan.focusWindow,
        tab_create_active: tabActivationPlan.createActive,
        tab_update_active: tabActivationPlan.updateActive,
        window_focus_policy: tabActivationPlan.focusWindow ? "focus_window" : "keep_window_background",
        capture_overlay: tabActivationMode !== "background_tab",
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

  const activateCollectionTab = shouldActivateCollectionTab(job, source);
  const captureOverlayText = collectorCaptureOverlayText(job, binding);
  const tabActivationPlan = collectionTabActivationPlan(job, source);
  const tab = await createTab({ url: source.url, active: tabActivationPlan.createActive });
  try {
    if (activateCollectionTab) {
      await activateTab(tab, tabActivationPlan);
    }
    await waitForTabLoad(tab.id, 25000);
    if (activateCollectionTab) {
      await installCollectorCaptureOverlayOnTab(tab, captureOverlayText);
    }
    await delay(randomDelayMs(settings, job.pacing || {}));
    // Inject the cleaning pipeline (filtering.js + helpers), then run the
    // scroll/capture orchestrator. Same data_point envelope as before; the text
    // fields now carry the cleaned extraction instead of the raw innerText blob.
    await withTimeout(
      chrome.scripting.executeScript({ target: { tabId: tab.id }, files: CAPTURE_FILES }),
      20000,
      "inject_capture_files_timeout_needs_site_access"
    );
    const discoveryMode = isDiscoveryCollection(job, source);
    const maxScrollSteps = maxScrollStepsForCollection(job, source);
    const collectOptions = {
      scrollSteps: Number(job.pacing?.scroll_steps || settings.scrollSteps || 5),
      maxScrollSteps,
      scrollMode: discoveryMode ? "discovery" : "standard",
      stopAfterNoMoveScrolls: discoveryMode ? 3 : 0,
      minDelaySeconds: Number(job.pacing?.min_delay_seconds || settings.minDelaySeconds || 5),
      maxDelaySeconds: Number(job.pacing?.max_delay_seconds || settings.maxDelaySeconds || 5)
    };
    const captureTimeoutMs = captureTimeoutMsForCollection(job, source, settings, collectOptions);
    const [result] = await withTimeout(chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: collectCleanPage,
      args: [collectOptions]
    }), captureTimeoutMs, "capture_timeout_needs_visible_collector_window_or_site_access");

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
        tab_activation_mode: tabActivationPlan.mode,
        window_focus_requested: tabActivationPlan.focusWindow,
        tab_create_active: tabActivationPlan.createActive,
        tab_update_active: tabActivationPlan.updateActive,
        window_focus_policy: tabActivationPlan.focusWindow ? "focus_window" : "keep_window_background",
        capture_overlay: activateCollectionTab,
        capture_overlay_text: activateCollectionTab ? captureOverlayText : "",
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

function collectionTabActivationPlan(job, source) {
  const active = shouldActivateCollectionTab(job, source);
  if (!active) {
    return {
      mode: "background_tab",
      createActive: false,
      updateActive: false,
      focusWindow: false
    };
  }

  const pacing = job && typeof job === "object" ? job.pacing || {} : {};
  const focusWindow = (
    explicitTrue(job?.focus_collection_window) ||
    explicitTrue(job?.focus_window) ||
    explicitTrue(pacing?.focus_collection_window) ||
    explicitTrue(pacing?.focus_window) ||
    explicitTrue(source?.focus_collection_window) ||
    explicitTrue(source?.focus_window)
  );

  return {
    mode: focusWindow ? "active_tab_with_window_focus" : "active_tab_no_window_focus",
    createActive: focusWindow,
    updateActive: true,
    focusWindow
  };
}

async function activateTab(tab, plan = {}) {
  if (!tab || typeof tab.id !== "number") return;
  try {
    if (plan.updateActive !== false) {
      await chrome.tabs.update(tab.id, { active: true });
    }
    if (plan.focusWindow && typeof tab.windowId === "number") {
      await chrome.windows.update(tab.windowId, { focused: true });
    }
  } catch (error) {
    // Capture can still proceed if activating is blocked. Metadata records
    // whether the run requested window focus or only tab activation.
  }
}

async function installCollectorCaptureOverlayOnTab(tab, text) {
  if (!tab || typeof tab.id !== "number") return false;
  try {
    await withTimeout(chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: installCollectorCaptureOverlay,
      args: [{ text }]
    }), 8000, "install_capture_overlay_timeout");
    return true;
  } catch (error) {
    return false;
  }
}

function collectorCaptureOverlayText(job, binding) {
  const clientName = String(
    job?.client_name ||
    job?.client_display_name ||
    binding?.client_name ||
    ""
  ).trim();
  if (clientName) {
    return `${clientName} - Solo Agency Collector`;
  }
  const displayName = String(
    binding?.extension_display_name ||
    job?.extension_display_name ||
    "Solo Agency Collector"
  ).replace(/\s*-\s*/g, " ").trim();
  return displayName || "Solo Agency Collector";
}

function installCollectorCaptureOverlay(options) {
  const overlayId = "solo-agency-collector-capture-overlay";
  const styleId = "solo-agency-collector-capture-overlay-style";
  const label = String(options && options.text ? options.text : "Solo Agency Collector");
  try {
    const existing = document.getElementById(overlayId);
    if (existing) existing.remove();
    if (!document.getElementById(styleId)) {
      const style = document.createElement("style");
      style.id = styleId;
      style.textContent = `
@keyframes soloAgencyCollectorRecordBlink {
  0%, 100% { opacity: 1; transform: scale(1); box-shadow: 0 0 0 0 rgba(255, 0, 0, 0.72); }
  45% { opacity: 0.36; transform: scale(0.82); box-shadow: 0 0 0 18px rgba(255, 0, 0, 0); }
}
	#${overlayId} {
	  position: fixed !important;
	  top: 18px !important;
	  right: 18px !important;
	  z-index: 2147483647 !important;
	  display: flex !important;
	  flex-direction: column !important;
	  align-items: stretch !important;
	  gap: 10px !important;
	  width: min(380px, calc(100vw - 28px)) !important;
	  max-height: min(52vh, 430px) !important;
	  padding: 12px !important;
	  border: 1px solid rgba(248, 113, 113, 0.36) !important;
	  border-radius: 10px !important;
	  background: linear-gradient(180deg, rgba(17, 24, 39, 0.96), rgba(7, 10, 18, 0.94)) !important;
	  color: #f8fafc !important;
	  box-shadow: 0 16px 44px rgba(0, 0, 0, 0.34), 0 0 0 1px rgba(255, 255, 255, 0.08) inset !important;
	  pointer-events: none !important;
	  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Arial, sans-serif !important;
	  line-height: 1.25 !important;
	  letter-spacing: 0 !important;
	  text-align: left !important;
	  backdrop-filter: blur(10px) saturate(1.15) !important;
	}
	#${overlayId} .solo-agency-collector-record-head {
	  display: flex !important;
	  align-items: flex-start !important;
	  gap: 10px !important;
	}
	#${overlayId} .solo-agency-collector-record-dot {
	  width: 18px !important;
	  height: 18px !important;
	  min-width: 18px !important;
	  margin-top: 2px !important;
	  border-radius: 999px !important;
	  background: radial-gradient(circle at 35% 35%, #fecaca 0, #ef4444 38%, #991b1b 100%) !important;
	  animation: soloAgencyCollectorRecordBlink 0.9s infinite !important;
	}
	#${overlayId} .solo-agency-collector-title-stack {
	  display: flex !important;
	  flex-direction: column !important;
	  gap: 4px !important;
	  min-width: 0 !important;
	}
	#${overlayId} .solo-agency-collector-record-label {
	  color: #ffffff !important;
	  font-size: 15px !important;
	  font-weight: 800 !important;
	  line-height: 1.2 !important;
	  letter-spacing: 0 !important;
	  white-space: nowrap !important;
	  overflow: hidden !important;
	  text-overflow: ellipsis !important;
	  max-width: 320px !important;
	}
	#${overlayId} .solo-agency-collector-record-subtitle {
	  color: #cbd5e1 !important;
	  font-size: 11px !important;
	  font-weight: 600 !important;
	  line-height: 1.3 !important;
	  letter-spacing: 0 !important;
	}
	#${overlayId} .solo-agency-collector-privacy {
	  color: #fecaca !important;
	  font-size: 10px !important;
	  font-weight: 700 !important;
	  line-height: 1.25 !important;
	  letter-spacing: 0 !important;
	}
	#${overlayId} .solo-agency-collector-stream-title {
	  display: flex !important;
	  justify-content: space-between !important;
	  align-items: center !important;
	  gap: 10px !important;
	  color: #e2e8f0 !important;
	  font-size: 11px !important;
	  font-weight: 800 !important;
	  letter-spacing: 0 !important;
	  text-transform: uppercase !important;
	}
	#${overlayId} .solo-agency-collector-stream-count {
	  color: #94a3b8 !important;
	  font-size: 10px !important;
	  font-weight: 700 !important;
	  text-transform: none !important;
	}
	#${overlayId} .solo-agency-collector-stream-box {
	  min-height: 92px !important;
	  max-height: min(30vh, 220px) !important;
	  overflow: hidden !important;
	  padding: 8px !important;
	  border: 1px solid rgba(148, 163, 184, 0.22) !important;
	  border-radius: 8px !important;
	  background: rgba(2, 6, 23, 0.62) !important;
	}
	#${overlayId} .solo-agency-collector-stream {
	  display: flex !important;
	  flex-direction: column !important;
	  gap: 6px !important;
	  max-height: min(28vh, 204px) !important;
	  overflow: hidden !important;
	  color: #dbeafe !important;
	  font-size: 11px !important;
	  font-weight: 500 !important;
	  line-height: 1.35 !important;
	  white-space: pre-wrap !important;
	}
	#${overlayId} .solo-agency-collector-stream-row {
	  padding: 6px 7px !important;
	  border-left: 2px solid rgba(248, 113, 113, 0.78) !important;
	  background: rgba(15, 23, 42, 0.72) !important;
	  border-radius: 6px !important;
	}
	#${overlayId} .solo-agency-collector-stream-meta {
	  display: block !important;
	  margin-bottom: 3px !important;
	  color: #fca5a5 !important;
	  font-size: 9px !important;
	  font-weight: 800 !important;
	  letter-spacing: 0 !important;
	  text-transform: uppercase !important;
	}
`;
      (document.head || document.documentElement).appendChild(style);
    }
    const overlay = document.createElement("div");
    overlay.id = overlayId;
    overlay.setAttribute("role", "status");
    overlay.setAttribute("aria-live", "polite");
    const head = document.createElement("div");
    head.className = "solo-agency-collector-record-head";
    const dot = document.createElement("span");
    dot.className = "solo-agency-collector-record-dot";
	    const text = document.createElement("span");
	    text.className = "solo-agency-collector-record-label";
	    text.textContent = label;
	    const stack = document.createElement("div");
	    stack.className = "solo-agency-collector-title-stack";
	    const subtitle = document.createElement("div");
	    subtitle.className = "solo-agency-collector-record-subtitle";
	    subtitle.textContent = "Capturing approved visible-page data locally";
	    const privacy = document.createElement("div");
	    privacy.className = "solo-agency-collector-privacy";
	    privacy.textContent = "No username, password, cookie, token, or account secret is collected or sent out.";
	    stack.appendChild(text);
	    stack.appendChild(subtitle);
	    stack.appendChild(privacy);
	    head.appendChild(dot);
	    head.appendChild(stack);
	    const streamTitle = document.createElement("div");
	    streamTitle.className = "solo-agency-collector-stream-title";
	    const streamTitleText = document.createElement("span");
	    streamTitleText.textContent = "Live data preview";
	    const streamCount = document.createElement("span");
	    streamCount.className = "solo-agency-collector-stream-count";
	    streamCount.textContent = "0 items";
	    streamTitle.appendChild(streamTitleText);
	    streamTitle.appendChild(streamCount);
    const streamBox = document.createElement("div");
    streamBox.className = "solo-agency-collector-stream-box";
    const stream = document.createElement("div");
    stream.className = "solo-agency-collector-stream";
    const initialRow = document.createElement("div");
    initialRow.className = "solo-agency-collector-stream-row";
    initialRow.textContent = "Waiting for visible page text...";
    stream.appendChild(initialRow);
    streamBox.appendChild(stream);
    overlay.appendChild(head);
    overlay.appendChild(streamTitle);
    overlay.appendChild(streamBox);
    (document.body || document.documentElement).appendChild(overlay);
    window.__soloAgencyCollectorUpdateOverlay = function(payload) {
      try {
	        const target = document.querySelector(`#${overlayId} .solo-agency-collector-stream`);
	        const box = document.querySelector(`#${overlayId} .solo-agency-collector-stream-box`);
	        const count = document.querySelector(`#${overlayId} .solo-agency-collector-stream-count`);
	        if (!target || !box) return;
	        const data = payload && typeof payload === "object" ? payload : {};
        const row = document.createElement("div");
        row.className = "solo-agency-collector-stream-row";
        const meta = document.createElement("span");
        meta.className = "solo-agency-collector-stream-meta";
        const step = data.step != null && data.total != null ? `step ${data.step}/${data.total}` : "collector";
        meta.textContent = `${step} - ${String(data.phase || "capturing")}`;
        const body = document.createElement("span");
        const raw = String(data.text || data.message || "").replace(/\s+/g, " ").trim();
	        body.textContent = raw ? raw.slice(-520) : "Collecting visible page text...";
	        row.appendChild(meta);
	        row.appendChild(body);
	        target.appendChild(row);
	        while (target.children.length > 8) target.removeChild(target.firstChild);
	        if (count) {
	          const itemCount = Math.max(0, target.children.length - 1);
	          count.textContent = `${itemCount} item${itemCount === 1 ? "" : "s"}`;
	        }
	        box.scrollTop = box.scrollHeight;
        target.scrollTop = target.scrollHeight;
      } catch (error) {
        // Overlay preview is informational only; never block collection.
      }
    };
    return { ok: true, text: label };
  } catch (error) {
    return { ok: false, error: String(error && error.message ? error.message : error) };
  }
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

async function withTimeout(promise, timeoutMs, label) {
  const original = Promise.resolve(promise);
  let timer = null;
  try {
    return await Promise.race([
      original,
      new Promise((_, reject) => {
        timer = setTimeout(() => {
          reject(new Error(`${label || "operation_timeout"} after ${Math.round((timeoutMs || 0) / 1000)}s`));
        }, timeoutMs || 30000);
      })
    ]);
  } finally {
    if (timer) clearTimeout(timer);
    original.catch(() => {});
  }
}

function captureTimeoutMsForCollection(job, source, settings, options) {
  const maxScrollSteps = maxScrollStepsForCollection(job, source);
  const scrollSteps = clampNumber(options?.scrollSteps, 0, maxScrollSteps, settings.scrollSteps || DEFAULT_SETTINGS.scrollSteps);
  const maxDelaySeconds = clampNumber(options?.maxDelaySeconds, 1, 60, settings.maxDelaySeconds || DEFAULT_SETTINGS.maxDelaySeconds);
  const perStepMs = (maxDelaySeconds * 1000) + 25000;
  const estimatedMs = 90000 + Math.max(1, scrollSteps) * perStepMs;
  const capMs = isDiscoveryCollection(job, source) ? 25 * 60 * 1000 : 12 * 60 * 1000;
  return Math.max(3 * 60 * 1000, Math.min(capMs, estimatedMs));
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

  function humanDurationMs(baseMs) {
    return Math.round(clamp(Number(baseMs) * randomBetween(0.72, 1.38), 420, 1450));
  }

  function humanScrollDistance(viewport, multiplier) {
    const base = Math.max(680, Math.floor((viewport || window.innerHeight || 800) * multiplier));
    return Math.round(base * randomBetween(0.84, 1.12));
  }

  function dispatchScrollHints(target, distance) {
    const node = target && target.kind === "element" ? target.el : document.scrollingElement || document.documentElement || document.body;
    try {
      const wheel = new WheelEvent("wheel", {
        bubbles: true,
        cancelable: true,
        deltaY: Math.max(120, Number(distance) || 0),
        view: window
      });
      (node || document).dispatchEvent(wheel);
    } catch (error) {
      // Synthetic input hints are best-effort only.
    }
    try {
      (target && target.kind === "element" ? target.el : window).dispatchEvent(new Event("scroll", { bubbles: true }));
    } catch (error) {
      // Ignore pages that block synthetic events.
    }
  }

  async function moveScrollTarget(target, toTop, durationMs) {
    const from = scrollTopOf(target);
    const max = scrollMaxOf(target);
    const to = Math.max(0, Math.min(max, Number(toTop) || 0));
    if (Math.abs(to - from) < 1) return;
    const duration = Math.max(260, Number(durationMs) || 760);
    if (Math.abs(to - from) > 240) {
      setScrollTopOf(target, from + (to - from) * 0.58);
      dispatchScrollHints(target, to - from);
      await wait(Math.round(clamp(duration / 7, 60, 180)));
    }
    setScrollTopOf(target, to);
    dispatchScrollHints(target, to - from);
  }

  async function tryScrollTarget(target, distance) {
    const before = scrollTopOf(target);
    const nextTop = Math.min(scrollMaxOf(target), before + distance);
    const durationMs = humanDurationMs(scrollMode === "discovery" ? 850 : 700);
    await moveScrollTarget(target, nextTop, durationMs);
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
    await moveScrollTarget(fallbackTarget, scrollTopOf(fallbackTarget) + fallbackDistance, durationMs);
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

  function compactOverlayText(value) {
    return String(value || "").replace(/\s+/g, " ").trim().slice(-1800);
  }

  function overlayTextFromCapture(capture) {
    if (!capture) return "";
    if (capture.error) return `Capture warning: ${String(capture.error)}`;
    const text = capture.mergedDisplay || capture.merged || capture.display || capture.text || "";
    return compactOverlayText(text);
  }

  function updateCollectorOverlay(payload) {
    try {
      if (typeof window.__soloAgencyCollectorUpdateOverlay === "function") {
        window.__soloAgencyCollectorUpdateOverlay(payload || {});
      }
    } catch (error) {
      // The on-page preview is informational only; never block collection.
    }
  }

  let prev = "";
  let last = null;
  let scrollStepsUsed = 0;
  let consecutiveNoMove = 0;
  let stoppedReason = "";
  updateCollectorOverlay({
    phase: "starting",
    step: 0,
    total: steps,
    text: `Preparing local capture for ${location.href}`
  });
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
    updateCollectorOverlay({
      phase: "capturing",
      step: Math.min(i, steps),
      total: steps,
      text: overlayTextFromCapture(last) || `Captured visible page state at ${location.href}`
    });
    if (i < steps) {
      const scrollReport = await performCollectorScroll();
      scrollStepsUsed += 1;
      scrollReport.pause_ms = delayMs();
      scrollDebug.push(scrollReport);
      updateCollectorOverlay({
        phase: "scrolling",
        step: scrollStepsUsed,
        total: steps,
        text: `Scrolled ${scrollReport.target || "page"} by ${scrollReport.delta == null ? "unknown" : scrollReport.delta} px. Waiting ${Math.round(scrollReport.pause_ms / 1000)}s for new visible data to load.`
      });
      if (!scrollReport || Math.abs(Number(scrollReport.delta || 0)) < 80) {
        consecutiveNoMove += 1;
      } else {
        consecutiveNoMove = 0;
      }
      if (stopAfterNoMoveScrolls > 0 && consecutiveNoMove >= stopAfterNoMoveScrolls) {
        stoppedReason = `no_scroll_movement_${consecutiveNoMove}_times`;
        updateCollectorOverlay({
          phase: "stopping",
          step: scrollStepsUsed,
          total: steps,
          text: `Stopping local capture: ${stoppedReason}.`
        });
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
  updateCollectorOverlay({
    phase: "completed",
    step: scrollStepsUsed,
    total: steps,
    text: "Local visible-page capture completed. Saving collected data to the Local Collector app on this computer."
  });
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

  function dispatchScrollHints(target, distance) {
    const node = target && target.kind === "element" ? target.el : document.scrollingElement || document.documentElement || document.body;
    try {
      const wheel = new WheelEvent("wheel", {
        bubbles: true,
        cancelable: true,
        deltaY: Math.max(120, Number(distance) || 0),
        view: window
      });
      (node || document).dispatchEvent(wheel);
    } catch (error) {
      // Synthetic input hints are best-effort only.
    }
    try {
      (target && target.kind === "element" ? target.el : window).dispatchEvent(new Event("scroll", { bubbles: true }));
    } catch (error) {
      // Ignore pages that block synthetic events.
    }
  }

  async function moveScrollTarget(target, toTop, durationMs) {
    const from = scrollTopOf(target);
    const max = scrollMaxOf(target);
    const to = Math.max(0, Math.min(max, Number(toTop) || 0));
    if (Math.abs(to - from) < 1) return;
    const duration = Math.max(260, Number(durationMs) || 850);
    if (Math.abs(to - from) > 240) {
      setScrollTopOf(target, from + (to - from) * 0.58);
      dispatchScrollHints(target, to - from);
      await wait(Math.round(clamp(duration / 7, 60, 180)));
    }
    setScrollTopOf(target, to);
    dispatchScrollHints(target, to - from);
  }

  async function tryScrollTarget(target, distance) {
    const before = scrollTopOf(target);
    const nextTop = Math.min(scrollMaxOf(target), before + distance);
    const durationMs = humanDurationMs(850);
    await moveScrollTarget(target, nextTop, durationMs);
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
    const hasSameRun = inMemoryActiveRuns.has(runId);
    if (!hasSameRun && shouldReplaceStaleRunLock(runId, job, bridgeStatus)) {
      inMemoryActiveRuns.clear();
    } else {
      return { acquired: false, reason: "in_memory_active_run" };
    }
  }

  const lockData = await chrome.storage.local.get(ACTIVE_RUN_KEY);
  if (inMemoryActiveRuns.size > 0) {
    const hasSameRun = inMemoryActiveRuns.has(runId);
    if (!hasSameRun && shouldReplaceStaleRunLock(runId, job, bridgeStatus)) {
      inMemoryActiveRuns.clear();
    } else {
      return { acquired: false, reason: "in_memory_active_run" };
    }
  }

  let currentLock = lockData[ACTIVE_RUN_KEY] || null;
  const lockExpiresAt = currentLock && Date.parse(currentLock.expiresAt || "");
  if (currentLock && Number.isFinite(lockExpiresAt) && lockExpiresAt > now) {
    if (currentLock.runId !== runId && shouldReplaceStaleRunLock(runId, job, bridgeStatus)) {
      await chrome.storage.local.remove(ACTIVE_RUN_KEY);
      currentLock = null;
    }
  }
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

function shouldReplaceStaleRunLock(runId, job, bridgeStatus) {
  if (!runId) return false;
  const bridgeRunId = String(bridgeStatus?.run_id || "");
  if (bridgeRunId && bridgeRunId !== runId) return false;
  return Boolean(job?.force || job?.run_now || job?.active_tab_diagnostic || job?.activate_tab_for_test);
}

async function releaseRunLock(runId, owner) {
  inMemoryActiveRuns.delete(runId);
  const lockData = await chrome.storage.local.get(ACTIVE_RUN_KEY);
  const currentLock = lockData[ACTIVE_RUN_KEY] || null;
  if (currentLock && currentLock.runId === runId && currentLock.owner === owner) {
    await chrome.storage.local.remove(ACTIVE_RUN_KEY);
  }
}

async function resetRunLockAfterBuildChange(reason) {
  const data = await chrome.storage.local.get(BUILD_STATE_KEY);
  if (data[BUILD_STATE_KEY] === EXTENSION_BUILD) return false;
  inMemoryActiveRuns.clear();
  await chrome.storage.local.remove(ACTIVE_RUN_KEY);
  await chrome.storage.local.set({ [BUILD_STATE_KEY]: EXTENSION_BUILD });
  await setState({
    status: "updated",
    message: `Collector extension updated to ${EXTENSION_BUILD}; stale run lock cleared.`,
    reason,
    extensionBuild: EXTENSION_BUILD,
    updatedAt: new Date().toISOString()
  });
  return true;
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

function shouldActivateCollectionTab(job, source) {
  const pacing = job && typeof job === "object" ? job.pacing || {} : {};
  if (
    explicitFalse(job?.activate_tab) ||
    explicitFalse(job?.focus_collection_tab) ||
    explicitFalse(pacing?.activate_tab) ||
    explicitFalse(pacing?.focus_collection_tab) ||
    explicitFalse(source?.activate_tab) ||
    explicitFalse(source?.focus_collection_tab) ||
    explicitTrue(job?.background_tab) ||
    explicitTrue(job?.allow_background_tab) ||
    explicitTrue(pacing?.background_tab) ||
    explicitTrue(pacing?.allow_background_tab) ||
    explicitTrue(source?.background_tab) ||
    explicitTrue(source?.allow_background_tab)
  ) {
    return false;
  }
  return true;
}

function explicitTrue(value) {
  if (value === true) return true;
  const normalized = String(value || "").trim().toLowerCase();
  return normalized === "true" || normalized === "1" || normalized === "yes" || normalized === "on";
}

function explicitFalse(value) {
  if (value === false) return true;
  const normalized = String(value || "").trim().toLowerCase();
  return normalized === "false" || normalized === "0" || normalized === "no" || normalized === "off";
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
