// Solo Agency Local Collector — operator alert player (runs in offscreen.html).
//
// WHY AN OFFSCREEN DOCUMENT. A MV3 service worker has no audio; a tab the collector opened
// has had no user gesture, so an AudioContext created inside it stays "suspended" (autoplay
// policy) and never sounds. Extension pages are exempt from that policy, and the offscreen
// document is the MV3 extension page that exists precisely for AUDIO_PLAYBACK. background.js
// creates it on demand and talks to it with runtime messages:
//   { type: "solo_operator_alert", action: "start", reason, detail }  -> begin the chime loop
//   { type: "solo_operator_alert", action: "stop" }                   -> silence
//   { type: "solo_operator_alert", action: "status" }                 -> { playing, since, reason }
//
// THE SOUND. Gentle by design (the operator asked for "nhẹ nhàng nhưng liên tục"): a soft
// two-note sine chime (E5 659 Hz -> A5 880 Hz, ~0.5 s) every ~3.2 s, peak gain 0.06, master
// fades in over the first 2 s so it never startles, and it repeats until stopped. Melody
// inherited from the operator's gubo-browser zillow module (880/1100 Hz beeps), softened.
(function () {
  var MSG_TYPE = "solo_operator_alert";
  var PERIOD_MS = 3200;
  var ctx = null, master = null, timer = null, playing = false, since = null, reason = "";

  function ensureContext() {
    if (ctx) return ctx;
    var AC = window.AudioContext || window.webkitAudioContext;
    if (!AC) return null;
    ctx = new AC();
    master = ctx.createGain();
    master.gain.value = 0;
    master.connect(ctx.destination);
    return ctx;
  }
  function note(freq, at, dur, peak) {
    var osc = ctx.createOscillator(), g = ctx.createGain();
    osc.type = "sine";
    osc.frequency.value = freq;
    g.gain.setValueAtTime(0.0001, at);
    g.gain.exponentialRampToValueAtTime(peak, at + 0.03);      // soft attack
    g.gain.exponentialRampToValueAtTime(0.0001, at + dur);     // long, gentle release
    osc.connect(g); g.connect(master);
    osc.start(at); osc.stop(at + dur + 0.02);
  }
  function chime() {
    if (!playing || !ctx) return;
    try {
      if (ctx.state === "suspended") ctx.resume();
      var t = ctx.currentTime + 0.02;
      note(659.25, t, 0.45, 0.06);        // E5
      note(880.0, t + 0.24, 0.55, 0.05);  // A5
    } catch (e) { /* keep looping; a single failed chime is not worth stopping the alert */ }
  }
  function start(why, detail) {
    if (!ensureContext()) return { ok: false, error: "no AudioContext" };
    reason = String(why || "operator_needed");
    if (playing) return { ok: true, playing: true, since: since, reason: reason, already: true };
    playing = true; since = new Date().toISOString();
    try {
      if (ctx.state === "suspended") ctx.resume();
      var now = ctx.currentTime;
      master.gain.cancelScheduledValues(now);
      master.gain.setValueAtTime(0.0001, now);
      master.gain.exponentialRampToValueAtTime(1, now + 2.0);   // fade the whole loop in
    } catch (e) { /* ignore */ }
    chime();
    timer = setInterval(chime, PERIOD_MS);
    return { ok: true, playing: true, since: since, reason: reason, detail: detail || null };
  }
  function stop() {
    if (timer) { clearInterval(timer); timer = null; }
    playing = false;
    try {
      if (ctx && master) {
        var now = ctx.currentTime;
        master.gain.cancelScheduledValues(now);
        master.gain.setValueAtTime(master.gain.value || 0.0001, now);
        master.gain.exponentialRampToValueAtTime(0.0001, now + 0.3);
      }
    } catch (e) { /* ignore */ }
    return { ok: true, playing: false, since: since, reason: reason };
  }
  function status() { return { ok: true, playing: playing, since: since, reason: reason }; }

  function handle(message) {
    if (!message || message.type !== MSG_TYPE) return null;
    if (message.action === "start") return start(message.reason, message.detail);
    if (message.action === "stop") return stop();
    return status();
  }

  if (typeof chrome !== "undefined" && chrome.runtime && chrome.runtime.onMessage) {
    chrome.runtime.onMessage.addListener(function (message, sender, sendResponse) {
      var res = handle(message);
      if (res === null) return false;         // not ours — let other listeners answer
      sendResponse(res);
      return false;
    });
  }
  // Exposed for the offline harness (tests/test_offscreen_alert.js).
  window.__soloOperatorAlert = { handle: handle, start: start, stop: stop, status: status, PERIOD_MS: PERIOD_MS, MSG_TYPE: MSG_TYPE };
})();
