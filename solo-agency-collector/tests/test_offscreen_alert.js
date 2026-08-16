// Offline harness for chrome-extension/offscreen.js — the operator alert player that
// background.js drives while a job waits for a human (Zillow "Press & Hold").
//
// Audio cannot be heard in Node, so a fake AudioContext records what would be scheduled: the
// test proves the message protocol (start / stop / status), that start is idempotent, that the
// chime keeps repeating until stop, that stop silences and clears the timer, and that the
// levels stay GENTLE (peak gain ≤ 0.06, master fades in) — the operator asked for "nhẹ nhàng
// nhưng liên tục".
//
// Run:  node solo-agency-collector/tests/test_offscreen_alert.js
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "offscreen.js"), "utf8");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

function fakeAudio() {
  const log = { oscillators: [], gains: [], resumes: 0 };
  function Param(initial) {
    const p = { value: initial, sets: [] };
    p.setValueAtTime = (v, t) => { p.sets.push(["set", v, t]); p.value = v; };
    p.exponentialRampToValueAtTime = (v, t) => { p.sets.push(["ramp", v, t]); p.value = v; };
    p.cancelScheduledValues = () => { p.sets.push(["cancel"]); };
    return p;
  }
  class Ctx {
    constructor() { this.state = "running"; this.currentTime = 0; this.destination = { id: "dest" }; }
    resume() { log.resumes += 1; this.state = "running"; return Promise.resolve(); }
    createGain() { const g = { gain: Param(1), connect(to) { g.to = to; }, }; log.gains.push(g); return g; }
    createOscillator() {
      const o = { type: "", frequency: Param(0), started: null, stopped: null, connect(to) { o.to = to; }, start(t) { o.started = t; }, stop(t) { o.stopped = t; } };
      log.oscillators.push(o); return o;
    }
  }
  return { Ctx, log };
}

function makeCtx() {
  const audio = fakeAudio();
  const timers = [];
  const ctx = {
    console,
    setInterval: (fn, ms) => { const id = { fn, ms, cleared: false }; timers.push(id); return id; },
    clearInterval: (id) => { if (id) id.cleared = true; },
    chrome: undefined,
  };
  ctx.window = ctx;
  ctx.window.AudioContext = audio.Ctx;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx, { filename: "offscreen.js" });
  return { ctx, audio, timers };
}

(function main() {
  console.log("protocol: start / status / idempotent start / stop");
  {
    const { ctx, audio, timers } = makeCtx();
    const A = ctx.window.__soloOperatorAlert;
    check("exposed for tests, message type + period", A && A.MSG_TYPE === "solo_operator_alert" && A.PERIOD_MS >= 2500 && A.PERIOD_MS <= 4000, A && [A.MSG_TYPE, A.PERIOD_MS]);
    check("foreign message is ignored (null)", A.handle({ type: "get_state" }) === null && A.handle(null) === null);
    const st0 = A.handle({ type: "solo_operator_alert", action: "status" });
    check("status before start: not playing", st0.ok === true && st0.playing === false, st0);
    const r1 = A.handle({ type: "solo_operator_alert", action: "start", reason: "zillow_bot_check", detail: { url: "x" } });
    check("start → playing, reason kept, one repeating timer at PERIOD_MS", r1.ok === true && r1.playing === true && r1.reason === "zillow_bot_check" && timers.length === 1 && timers[0].ms === A.PERIOD_MS && !timers[0].cleared, [r1, timers.map((t) => t.ms)]);
    check("first chime played immediately: 2 oscillators (E5 659.25 then A5 880), sine", audio.log.oscillators.length === 2 && audio.log.oscillators[0].frequency.value === 659.25 && audio.log.oscillators[1].frequency.value === 880 && audio.log.oscillators.every((o) => o.type === "sine"), audio.log.oscillators.map((o) => [o.type, o.frequency.value]));
    const master = audio.log.gains[0];
    check("master gain fades in from ~0 to 1 over ~2s (no startle)", master.gain.sets.some((s) => s[0] === "ramp" && s[1] === 1 && s[2] >= 1.5), master.gain.sets);
    const noteGains = audio.log.gains.slice(1);
    const peaks = noteGains.map((g) => Math.max.apply(null, g.gain.sets.filter((s) => s[0] === "ramp").map((s) => s[1])));
    check("note peaks are gentle (≤ 0.06) with a release ramp back to ~0", peaks.every((p) => p <= 0.06 && p > 0) && noteGains.every((g) => g.gain.sets.some((s) => s[0] === "ramp" && s[1] < 0.001)), peaks);
    const r2 = A.handle({ type: "solo_operator_alert", action: "start", reason: "zillow_bot_check" });
    check("second start is a no-op (already:true, still one timer)", r2.ok === true && r2.already === true && timers.length === 1, [r2, timers.length]);
    // the timer keeps chiming while playing
    const before = audio.log.oscillators.length;
    timers[0].fn(); timers[0].fn();
    check("each tick plays another chime (2 oscillators per tick)", audio.log.oscillators.length === before + 4, audio.log.oscillators.length);
    const st1 = A.handle({ type: "solo_operator_alert", action: "status" });
    check("status while playing", st1.playing === true && typeof st1.since === "string", st1);
    const r3 = A.handle({ type: "solo_operator_alert", action: "stop" });
    check("stop → not playing, timer cleared, master ramps to silence", r3.ok === true && r3.playing === false && timers[0].cleared === true && master.gain.sets.some((s) => s[0] === "ramp" && s[1] < 0.001), [r3, timers[0].cleared]);
    const after = audio.log.oscillators.length;
    timers[0].fn();
    check("a stray tick after stop plays nothing", audio.log.oscillators.length === after);
    const r4 = A.handle({ type: "solo_operator_alert", action: "start", reason: "again" });
    check("start after stop works again with a fresh timer", r4.playing === true && timers.length === 2 && !timers[1].cleared, [r4, timers.length]);
    A.handle({ type: "solo_operator_alert", action: "stop" });
  }

  console.log("suspended context: start resumes it, chime still scheduled");
  {
    const { ctx, audio } = makeCtx();
    // make the next context start suspended
    const OrigCtx = audio.Ctx;
    ctx.window.AudioContext = class extends OrigCtx { constructor() { super(); this.state = "suspended"; } };
    const A = ctx.window.__soloOperatorAlert;
    const r = A.handle({ type: "solo_operator_alert", action: "start", reason: "t" });
    check("start on a suspended context calls resume() and plays", r.playing === true && audio.log.resumes >= 1 && audio.log.oscillators.length === 2, [audio.log.resumes, audio.log.oscillators.length]);
    A.handle({ type: "solo_operator_alert", action: "stop" });
  }

  console.log("no AudioContext at all: start reports ok:false, never throws");
  {
    const { ctx } = makeCtx();
    ctx.window.AudioContext = undefined; ctx.window.webkitAudioContext = undefined;
    const A = ctx.window.__soloOperatorAlert;
    let threw = false, r = null;
    try { r = A.handle({ type: "solo_operator_alert", action: "start" }); } catch (e) { threw = true; }
    check("graceful failure", !threw && r && r.ok === false && /AudioContext/.test(r.error), r);
  }

  console.log("");
  console.log(pass + " passed, " + fail + " failed");
  process.exit(fail ? 1 : 0);
})();
