const DEFAULT_SETTINGS = {
  enabled: true,
  bridgeBaseUrl: "http://127.0.0.1:17321",
  pollMinutes: 1,
  pollSeconds: 5,
  minDelaySeconds: 5,
  maxDelaySeconds: 5,
  maxSourcesPerRun: 20,
  scrollSteps: 5,
  maxTextChars: 12000,
  closeTabsAfterCollect: true
};

const STATE_KEY = "collector_state";
const SETTINGS_KEY = "collector_settings";
const COMPLETED_RUNS_KEY = "collector_completed_runs";
const ACTIVE_RUN_KEY = "collector_active_run";
const ACTIVE_RUN_LOCK_MINUTES = 120;
const EXTENSION_BUILD = "0.1.1-run-lock";

const inMemoryActiveRuns = new Set();

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
        message: bridgeConfigSaved ? "Settings saved locally and to bridge." : "Settings saved locally. Bridge is offline.",
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
  if (!settings.enabled) {
    await setState({ status: "disabled", message: "Collector disabled.", reason });
    return { status: "disabled" };
  }

  const bridgeBaseUrl = trimSlash(settings.bridgeBaseUrl);
  let status;
  try {
    status = await fetchJSON(`${bridgeBaseUrl}/status`, {
      method: "GET",
      headers: {
        "X-Collector-Extension": "media-agency-local-collector",
        "X-Collector-Extension-Version": EXTENSION_BUILD
      }
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

  const job = await fetchJSON(`${bridgeBaseUrl}/jobs/current`, { method: "GET" }, 8000);
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
    await runJob({ job, token, bridgeBaseUrl, settings, reason });
    completedRuns[runId] = new Date().toISOString();
    await chrome.storage.local.set({ [COMPLETED_RUNS_KEY]: trimCompletedRuns(completedRuns) });
    return { status: "completed", runId };
  } finally {
    await releaseRunLock(runId, runLock.owner);
  }
}

async function runJob({ job, token, bridgeBaseUrl, settings, reason }) {
  const runId = String(job.run_id || job.collector_bridge?.run_id || "");
  const sources = normalizeSources(job.sources || []);
  const pacing = job.pacing || {};
  const maxSources = Math.min(
    Number(pacing.max_sources || settings.maxSourcesPerRun || 20),
    sources.length
  );
  const selectedSources = sources.slice(0, maxSources);
  const counts = {
    dataPoints: 0,
    competitors: 0,
    newPrivateSources: 0
  };

  await setState({
    status: "running",
    message: `Collecting ${selectedSources.length} private sources.`,
    runId,
    totalSources: selectedSources.length,
    dataPointsCollected: counts.dataPoints,
    competitorsDetected: counts.competitors,
    newPrivateSourcesDetected: counts.newPrivateSources,
    reason,
    updatedAt: new Date().toISOString()
  });

  for (let index = 0; index < selectedSources.length; index += 1) {
    const source = selectedSources[index];
    const sourceLabel = source.name || source.url || `source ${index + 1}`;
    await postToBridge(bridgeBaseUrl, token, "/collect/source_status", {
      run_id: runId,
      source_name: source.name || "",
      source_url: source.url || "",
      platform: source.platform || "",
      status: "started",
      index: index + 1,
      total: selectedSources.length,
      captured_at: new Date().toISOString(),
      collector_identity: "chrome-extension-local-collector"
    });

    await setState({
      status: "running",
      message: `Collecting ${index + 1}/${selectedSources.length}: ${sourceLabel}`,
      runId,
      currentSource: sourceLabel,
      currentScroll: 0,
      maxScrolls: Number(job.pacing?.scroll_steps || settings.scrollSteps || 5),
      dataPointsCollected: counts.dataPoints,
      competitorsDetected: counts.competitors,
      newPrivateSourcesDetected: counts.newPrivateSources,
      updatedAt: new Date().toISOString()
    });

    await delay(randomDelayMs(settings, pacing));

    try {
      const collected = await collectSource(source, job, settings);
      await postToBridge(bridgeBaseUrl, token, "/collect/data_point", collected.dataPoint);
      counts.dataPoints += 1;
      if (isCompetitorSource(source)) {
        await postToBridge(bridgeBaseUrl, token, "/collect/competitor", {
          run_id: runId,
          client_slug: job.client_slug || "",
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
        });
        counts.competitors += 1;
      }
      for (const candidate of collected.newPrivateSources || []) {
        await postToBridge(bridgeBaseUrl, token, "/collect/new_private_source", candidate);
        counts.newPrivateSources += 1;
      }
      await postToBridge(bridgeBaseUrl, token, "/collect/snapshot", collected.snapshot);
      await postToBridge(bridgeBaseUrl, token, "/collect/source_status", {
        run_id: runId,
        source_name: source.name || "",
        source_url: source.url || "",
        platform: source.platform || "",
        status: "collected",
        index: index + 1,
        total: selectedSources.length,
        current_url: collected.dataPoint.current_url || "",
        post_url: collected.dataPoint.post_url || collected.dataPoint.current_url || "",
        profile_url: collected.dataPoint.profile_url || "",
        captured_at: new Date().toISOString(),
        collector_identity: "chrome-extension-local-collector"
      });
      await setState({
        status: "running",
        message: `Collected ${index + 1}/${selectedSources.length}: ${sourceLabel}`,
        runId,
        currentSource: sourceLabel,
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
        source_name: source.name || "",
        source_url: source.url || "",
        platform: source.platform || "",
        status: "error",
        issue: String(error && error.message ? error.message : error),
        index: index + 1,
        total: selectedSources.length,
        captured_at: new Date().toISOString(),
        collector_identity: "chrome-extension-local-collector"
      });
    }
  }

  await postToBridge(bridgeBaseUrl, token, "/complete", {
    run_id: runId,
    status: "completed",
    completed_at: new Date().toISOString(),
    collector_identity: "chrome-extension-local-collector"
  });

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

async function collectSource(source, job, settings) {
  if (!source.url || !/^https?:\/\//i.test(source.url)) {
    throw new Error("source url must start with http:// or https://");
  }

  const tab = await createTab({ url: source.url, active: false });
  try {
    await waitForTabLoad(tab.id, 25000);
    await delay(randomDelayMs(settings, job.pacing || {}));
    const [result] = await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: collectVisiblePage,
      args: [source, job, {
        scrollSteps: Number(job.pacing?.scroll_steps || settings.scrollSteps || 5),
        minDelaySeconds: Number(job.pacing?.min_delay_seconds || settings.minDelaySeconds || 5),
        maxDelaySeconds: Number(job.pacing?.max_delay_seconds || settings.maxDelaySeconds || 5),
        maxTextChars: Number(job.pacing?.max_text_chars || settings.maxTextChars || 12000)
      }]
    });

    const page = result && result.result ? result.result : {};
    const now = new Date().toISOString();
    const runId = String(job.run_id || "");
    const snapshotFilename = `${safeSlug(runId || "run")}_${safeSlug(source.name || source.platform || "source")}_${Date.now()}.html`;

    return {
      dataPoint: {
        client_slug: job.client_slug || "",
        business_slug: job.business_slug || "",
        location_slug: job.location_slug || "",
        run_id: runId,
        source_name: source.name || "",
        source_url: source.url || "",
        source_type: source.source_type || source.type || "private",
        platform: source.platform || inferPlatform(source.url),
        priority: source.priority || "",
        scan_cadence: source.scan_cadence || "",
        purpose: source.purpose || "",
        current_url: page.current_url || "",
        post_url: page.post_url || page.current_url || "",
        profile_url: page.profile_url || "",
        profile_candidates: page.profile_candidates || [],
        post_candidates: page.post_candidates || [],
        title: page.title || "",
        visible_text_summary: page.visible_text_summary || "",
        raw_visible_text_excerpt: page.raw_visible_text_excerpt || "",
        engagement_hint: page.engagement_hint || "",
        captured_at: now,
        source_login_status: page.login_hint || "unknown",
        collector_identity: "chrome-extension-local-collector",
        confidence: page.visible_text_summary ? "medium" : "low",
        scroll_count: page.scroll_count || 0,
        max_scrolls: page.max_scrolls || 0,
        read_only: true
      },
      newPrivateSources: (page.new_private_source_candidates || []).map((candidate) => ({
        run_id: runId,
        client_slug: job.client_slug || "",
        platform: source.platform || inferPlatform(source.url),
        source_type: candidate.source_type || "group",
        source_name: candidate.text || candidate.href || "",
        profile_or_group_url: candidate.href || "unavailable",
        current_recommendation_url: page.current_url || "unavailable",
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
        filename: snapshotFilename,
        source_name: source.name || "",
        source_url: source.url || "",
        current_url: page.current_url || "",
        captured_at: now,
        html: buildSnapshotHTML({ source, job, page, capturedAt: now })
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

async function postToBridge(baseUrl, token, path, payload) {
  return fetchJSON(`${trimSlash(baseUrl)}${path}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Collector-Token": token,
      "X-Collector-Extension-Version": EXTENSION_BUILD
    },
    body: JSON.stringify(payload || {})
  }, 12000);
}

async function syncSettingsToBridge(settings) {
  const baseUrl = trimSlash(settings.bridgeBaseUrl || DEFAULT_SETTINGS.bridgeBaseUrl);
  try {
    let current = {};
    try {
      current = await fetchJSON(`${baseUrl}/config`, { method: "GET" }, 4000);
    } catch (error) {
      current = {};
    }
    const nextConfig = {
      ...current,
      version: current.version || "0.1.0",
      timezone: current.timezone || "local",
      run_mode: "persistent_bridge_scheduler",
      default_runs_per_day: current.default_runs_per_day || 1,
      poll_interval_seconds: clampNumber(settings.pollSeconds, 5, 60, 5),
      max_sources_per_run: clampNumber(settings.maxSourcesPerRun, 1, 20, 20),
      max_scrolls_per_source: clampNumber(settings.scrollSteps, 0, 10, 5),
      max_scrolls_allowed: 10,
      scroll_delay_seconds: clampNumber(settings.minDelaySeconds, 5, 60, 5),
      duplicate_filter: current.duplicate_filter || {
        compare_against_previous_day: true,
        method: "visible_text_matching",
        parse_html: false
      },
      scheduled_windows: current.scheduled_windows || [{
        name: "daily_default",
        enabled: true,
        local_time_start: "09:00",
        local_time_end: "09:30",
        days: ["mon", "tue", "wed", "thu", "fri", "sat", "sun"]
      }],
      clients: current.clients || []
    };
    await fetchJSON(`${baseUrl}/config`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(nextConfig)
    }, 4000);
    return true;
  } catch (error) {
    return false;
  }
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
  next.scrollSteps = clampNumber(next.scrollSteps, 0, 10, 5);
  next.maxTextChars = clampNumber(next.maxTextChars, 1000, 30000, 12000);
  next.closeTabsAfterCollect = Boolean(next.closeTabsAfterCollect);
  return next;
}

async function getSettings() {
  const data = await chrome.storage.local.get(SETTINGS_KEY);
  return normalizeSettings(data[SETTINGS_KEY] || DEFAULT_SETTINGS);
}

async function getState() {
  const data = await chrome.storage.local.get(STATE_KEY);
  return data[STATE_KEY] || { status: "idle", message: "Collector idle." };
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
  const scrollSteps = clampNumber(pacing.scroll_steps, 0, 10, DEFAULT_SETTINGS.scrollSteps);
  const maxDelaySeconds = clampNumber(pacing.max_delay_seconds, 5, 30, DEFAULT_SETTINGS.maxDelaySeconds);
  const estimatedMs = maxSources * ((scrollSteps + 2) * maxDelaySeconds * 1000 + 45000);
  const fallbackMs = ACTIVE_RUN_LOCK_MINUTES * 60 * 1000;
  return Math.max(30 * 60 * 1000, Math.min(fallbackMs, estimatedMs || fallbackMs));
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
