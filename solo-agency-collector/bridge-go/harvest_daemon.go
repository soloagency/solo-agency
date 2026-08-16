package main

// harvest_daemon.go — the mechanical half of Leads From Friends, running inside
// the persistent bridge (same shape as reply_poller.go). Every tick it looks at
// each client's active friend_harvest campaigns and does at most ONE action per
// campaign: collect finished jobs, enqueue one enrich, or enqueue one friends
// leg. Pacing, collector rotation, budgets, quiet hours and the circuit breaker
// are all enforced HERE, in code — an agent never sleeps 20-40 seconds
// reliably across a ten-hour session; a daemon does.
//
// Division of labour: the daemon walks and enriches; the AGENT decides.
// Enriched records land in {campaign}/enriched/, `harvest pending` hands them
// to the agent in small batches, `harvest decide` records the verdict.
//
// Fault tolerance (the operator's explicit requirement — long, multi-node):
//   - collector rotation uses the operator-wide harvest ledger: per-box daily
//     cap, per-box pacing gap and a circuit breaker (3 consecutive failures →
//     2h quarantine); a benched or dead box is simply not picked — the others
//     carry on, that is the failover;
//   - a job that goes stale (never claimed / never completed) is CANCELLED
//     (pending file removed when still unclaimed) and its friend is re-queued
//     at the front with attempts+1 and avoid_box=that box; only after
//     harvestMaxAttempts on different boxes does it become enrich_failed;
//   - a friends leg is exhausted ONLY when the connection says so; an empty or
//     errored leg keeps the cursor and is retried on another box; a seed no box
//     can read is flagged, never silently "done";
//   - seeds are reconciled with the campaign config every tick (added seeds
//     appear, removed seeds stop being walked, cursor kept for re-add);
//   - all state is on disk under flock — the daemon dying mid-tick loses nothing.

import (
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	harvestTickInterval  = 15 * time.Second
	harvestLegBaseWait   = 2 * time.Minute // extra rest before another leg on the SAME box
	harvestJobStaleAfter = 20 * time.Minute
)

var harvestRunning sync.Mutex

// harvestSleepGap: if the daemon's own ticker skips this long, the machine
// was asleep (laptop lid, no wifi). Failures recorded around that gap were
// the machine's, not the accounts' — so on wake the breaker forgives them.
const harvestSleepGap = 3 * time.Minute

func (b *bridge) startHarvestDaemon(stop <-chan struct{}) {
	go func() {
		select {
		case <-stop:
			return
		case <-time.After(45 * time.Second):
		}
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		ticker := time.NewTicker(harvestTickInterval)
		defer ticker.Stop()
		lastTick := time.Now()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				now := time.Now()
				if now.Sub(lastTick) > harvestSleepGap {
					b.harvestWakeAmnesty(now, lastTick)
				}
				lastTick = now
				b.harvestTick(rng, now)
			}
		}
	}()
}

// harvestWakeAmnesty runs once after the machine wakes: every quarantine
// caused by machine-side symptoms (stale jobs, no_record, source error —
// what a sleeping Chrome produces) is lifted and the failure counter reset,
// so the walk resumes immediately instead of waiting out a 2h bench that was
// never the account's fault. Genuine account symptoms (landed_on_self, an
// explicit checkpoint/error string) are NOT forgiven.
func (b *bridge) harvestWakeAmnesty(now, sleptAt time.Time) {
	lifted := 0
	_, _ = withLedger(b.uiDataRoot, now, func(l *harvestLedger) error {
		for id, bx := range l.Boxes {
			if !bx.quarantined(now) && bx.ConsecutiveFailures == 0 {
				continue
			}
			if !isMachineSideFailure(bx.LastError) {
				continue
			}
			bx.QuarantinedUntil = ""
			bx.ConsecutiveFailures = 0
			bx.LastError = "cleared: machine was asleep " + sleptAt.Format("15:04") + "–" + now.Format("15:04")
			lifted++
			_ = id
		}
		return nil
	})
	if lifted > 0 {
		log.Printf("harvest: machine woke after %s asleep — lifted quarantine on %d collector(s)", now.Sub(sleptAt).Round(time.Minute), lifted)
	}
}

// isMachineSideFailure: symptoms a sleeping/offline machine produces, as
// opposed to a Facebook account being restricted.
func isMachineSideFailure(reason string) bool {
	r := strings.ToLower(reason)
	if strings.Contains(r, "landed_on_self") || strings.Contains(r, "checkpoint") || strings.Contains(r, "restricted") {
		return false
	}
	for _, k := range []string{"stale", "never claimed", "no_record", "source error", "timeout", "capture", "navigation", "no result"} {
		if strings.Contains(r, k) {
			return true
		}
	}
	return false
}

type harvestCollector struct {
	ClientSlug string
	InstanceID string
	Name       string
}

// liveCollectors lists extensions that checked in recently, sorted for a stable
// round-robin.
func (b *bridge) liveCollectors(now time.Time) []harvestCollector {
	pollSeconds := getInt(b.configDoc, "poll_interval_seconds", 5)
	staleAfter := time.Duration(pollSeconds*3+10) * time.Second
	if staleAfter < 25*time.Second {
		staleAfter = 25 * time.Second
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []harvestCollector
	for _, ext := range b.extensions {
		if ext.lastCheckAt.IsZero() || now.Sub(ext.lastCheckAt) > staleAfter {
			continue
		}
		if ext.instanceID == "" || ext.clientSlug == "" {
			continue
		}
		out = append(out, harvestCollector{ClientSlug: ext.clientSlug, InstanceID: ext.instanceID, Name: ext.displayName})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].InstanceID < out[j].InstanceID })
	return out
}

// pickCollector rotates over live boxes starting after lastBox, returning the
// first one the ledger considers eligible (not quarantined, under cap, past
// its own pacing gap) and not equal to avoidBox.
func pickCollector(live []harvestCollector, lastBox, avoidBox string, ledger *harvestLedger, now time.Time, cap int, gap time.Duration) (harvestCollector, bool) {
	if len(live) == 0 {
		return harvestCollector{}, false
	}
	start := 0
	for i, c := range live {
		if c.InstanceID == lastBox {
			start = i + 1
			break
		}
	}
	for k := 0; k < len(live); k++ {
		c := live[(start+k)%len(live)]
		if c.InstanceID == avoidBox && len(live) > 1 {
			continue
		}
		if ledger.Boxes[c.InstanceID].eligible(now, cap, gap) {
			return c, true
		}
	}
	return harvestCollector{}, false
}

func (b *bridge) harvestTick(rng *rand.Rand, now time.Time) {
	if !harvestRunning.TryLock() {
		return
	}
	defer harvestRunning.Unlock()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("harvest: tick recovered: %v", r)
		}
	}()
	settings := loadSystemSettings(b.uiDataRoot)
	live := b.liveCollectors(now)
	for _, c := range b.uiClients() {
		outreachDir := filepath.Join(c.Path, "outreach")
		store := newCrmStore(outreachDir)
		for _, cfg := range store.listCampaigns() {
			camp := mStr(cfg, "campaign_slug")
			if camp == "" || mStr(cfg, "channel_strategy") != harvestChannel || mStr(cfg, "status") != "active" {
				continue
			}
			hc := harvestConfigFrom(cfg, settings)
			if inQuietHours(now, hc.QuietFrom, hc.QuietTo) {
				continue
			}
			b.harvestStep(rng, now, c, outreachDir, camp, hc, live)
		}
	}
}

// reconcileSeeds makes progress.Seeds mirror the campaign config: new seeds
// appended (cursor-less), seeds no longer configured flagged Removed (cursor
// kept so re-adding resumes). Idempotent, runs every tick.
func reconcileSeeds(p *harvestProgress, hc harvestConfig, clientDir, campaign string) {
	want := map[string]string{} // uid -> clean url
	for _, raw := range hc.SeedProfiles {
		clean, ok := canonicalStoreURL(raw)
		if !ok {
			continue
		}
		if uid, _ := sourceUID(clean); uid != "" {
			want[uid] = clean
		}
	}
	have := map[string]bool{}
	for i := range p.Seeds {
		s := &p.Seeds[i]
		have[s.UID] = true
		_, still := want[s.UID]
		s.Removed = !still
	}
	for uid, clean := range want {
		if have[uid] {
			continue
		}
		if fu := friendsURL(clean); fu != "" {
			p.Seeds = append(p.Seeds, harvestSeed{URL: clean, UID: uid, FriendsURL: fu})
		}
	}
	// Seeds are "seen" client-wide — never enrich a seed as if it were a friend.
	_, _ = withSeen(clientDir, func(reg *seenRegistry) error {
		now := nowISO()
		for _, s := range p.Seeds {
			if _, ok := reg.Profiles[s.UID]; !ok {
				reg.Profiles[s.UID] = seenProfile{UID: s.UID, URL: s.URL, Status: "seed",
					Campaign: campaign, FirstSeen: now, UpdatedAt: now}
			}
		}
		return nil
	})
	if p.CurrentSeed == "" {
		for _, s := range p.Seeds {
			if !s.Exhausted && !s.Removed && s.Error == "" {
				p.CurrentSeed = s.URL
				break
			}
		}
	}
}

// harvestStep advances one campaign by at most one action.
func (b *bridge) harvestStep(rng *rand.Rand, now time.Time, c uiClient, outreachDir, campaign string, hc harvestConfig, live []harvestCollector) {
	// 1. Collect finished / stale jobs first.
	b.harvestCollectResults(now, c, outreachDir, campaign, hc)

	// 2. Decide the next action from progress + ledger, atomically.
	var action string
	var seed harvestSeed
	var next harvestQueued
	var box harvestCollector
	var flight harvestInFlight
	gap := randomGap(rng)
	ledger, lerr := withLedger(b.uiDataRoot, now, func(l *harvestLedger) error {
		_, err := withProgress(outreachDir, campaign, func(p *harvestProgress) error {
			p.resetDayIfNeeded(now)
			reconcileSeeds(p, hc, outreachDir, campaign)
			if len(p.InFlight) > 0 {
				action = "wait" // one job at a time per campaign; stale ones are handled in collect
				return nil
			}
			if p.DayEnriched >= hc.DailyBudget {
				action = "budget"
				return nil
			}
			// Enrich queue first: friends already discovered are cheaper to finish.
			if len(p.Queue) > 0 {
				var ok bool
				box, ok = pickCollector(live, p.LastBox, p.Queue[0].AvoidBox, l, now, hc.PerBoxBudget, gap)
				if !ok {
					action = "no_collector"
					return nil
				}
				next = p.Queue[0]
				p.Queue = p.Queue[1:]
				runID := "harvest_enrich_" + now.UTC().Format("20060102_150405") + "_" + uidHash(next.UID)
				flight = harvestInFlight{URL: next.URL, Seed: next.Seed, RunID: runID, Box: box.InstanceID,
					QueuedAt: nowISO(), Name: next.Name, Subtitle: next.Subtitle, Priority: next.Priority, Attempts: next.Attempts}
				p.InFlight[next.UID] = flight
				p.LastEnrichAt = nowISO()
				p.LastBox = box.InstanceID
				p.DayEnriched++
				p.Totals["enrich_jobs"]++
				l.box(box.InstanceID).recordJob(now, "enrich")
				action = "enrich"
				return nil
			}
			// Queue empty: read the next leg of the first walkable seed.
			for i := range p.Seeds {
				s := &p.Seeds[i]
				if s.Exhausted || s.Removed || s.Error != "" {
					continue
				}
				// Never hand a seed back to a box that returned nothing for it.
				avoid := ""
				if len(s.TriedBoxes) > 0 {
					avoid = s.TriedBoxes[len(s.TriedBoxes)-1]
				}
				var ok bool
				box, ok = pickCollector(live, s.LastLegBox, avoid, l, now, hc.PerBoxBudget, gap)
				if !ok {
					action = "no_collector"
					return nil
				}
				if box.InstanceID == s.LastLegBox && s.LastLegAt != "" {
					if t, err := time.Parse(time.RFC3339, s.LastLegAt); err == nil && now.Sub(t) < harvestLegBaseWait {
						action = "leg_rest"
						return nil
					}
				}
				seed = *s
				p.CurrentSeed = s.URL
				runID := "harvest_leg_" + now.UTC().Format("20060102_150405") + "_" + uidHash(s.UID)
				flight = harvestInFlight{URL: s.FriendsURL, Seed: s.URL, RunID: runID, Box: box.InstanceID, QueuedAt: nowISO()}
				p.InFlight["leg:"+s.UID] = flight
				p.Totals["leg_jobs"]++
				l.box(box.InstanceID).recordJob(now, "leg")
				action = "leg"
				return nil
			}
			action = "done"
			return nil
		})
		return err
	})
	if lerr != nil {
		log.Printf("harvest[%s/%s]: %v", c.Slug, campaign, lerr)
		return
	}
	_ = ledger

	switch action {
	case "enrich":
		path := b.harvestEnqueue(now, c.Slug, campaign, box, flight.RunID, "fb.profile.enrich", next.URL,
			map[string]any{"profile_url": next.URL, "max_posts": 5}, next.UID, next.Seed)
		b.harvestRememberPath(outreachDir, campaign, next.UID, path)
	case "leg":
		inputs := map[string]any{"profile_url": seed.URL, "max_pages": hc.LegPages}
		if seed.EndCursor != "" {
			inputs["start_cursor"] = seed.EndCursor
		}
		path := b.harvestEnqueue(now, c.Slug, campaign, box, flight.RunID, "fb.profile.friends", seed.FriendsURL, inputs, "leg:"+seed.UID, seed.URL)
		b.harvestRememberPath(outreachDir, campaign, "leg:"+seed.UID, path)
	}
}

func (b *bridge) harvestRememberPath(outreachDir, campaign, tag, path string) {
	if path == "" {
		return
	}
	_, _ = withProgress(outreachDir, campaign, func(p *harvestProgress) error {
		if f, ok := p.InFlight[tag]; ok {
			f.PendingPath = path
			p.InFlight[tag] = f
		}
		return nil
	})
}

// harvestEnqueue writes one collector job bound to the chosen collector's
// extension, tagged so the result is routed to the OWNING client's harvest
// dir. Returns the pending-file path (for cancellation while unclaimed).
func (b *bridge) harvestEnqueue(now time.Time, ownerSlug, campaign string, box harvestCollector, runID, capability, url string, inputs map[string]any, tag, seed string) string {
	job := map[string]any{
		"run_id":                         runID,
		"job_type":                       "run_now",
		"client_slug":                    box.ClientSlug,
		"allowed_extension_instance_ids": []any{box.InstanceID},
		"extension_instance_id":          box.InstanceID,
		"harvest_owner":                  ownerSlug,
		"harvest_campaign":               campaign,
		"harvest_tag":                    tag,
		"harvest_seed":                   seed,
		"run_now_ttl_minutes":            15,
		"sources": []any{map[string]any{
			"name": "harvest " + capability, "url": url, "platform": "facebook",
			"source_type": "harvest", "capability": capability, "inputs": inputs, "priority": "high",
		}},
		"pacing":           map[string]any{"min_delay_seconds": 3, "max_delay_seconds": 6, "max_sources": 1, "scroll_steps": 0, "max_text_chars": 12000},
		"collector_policy": defaultCollectorPolicy(),
	}
	resp, err := b.enqueueRunNowPayload(job, now, "harvest")
	if err != nil {
		log.Printf("harvest[%s/%s]: enqueue %s: %v", ownerSlug, campaign, capability, err)
		return ""
	}
	if paths := asSlice(resp["queued_paths"]); len(paths) > 0 {
		return getString(map[string]any{"p": paths[0]}, "p", "")
	}
	return ""
}

// harvestCollectResults folds finished jobs in and handles stale ones:
//   - finished leg     → ingestLeg (tri-state outcome), ledger success/failure
//   - finished enrich  → enriched/{uid_hash}.json (+ok/error), await_decision
//   - stale job        → cancel pending file if unclaimed, ledger failure,
//                        re-queue the friend (or give up after max attempts)
func (b *bridge) harvestCollectResults(now time.Time, c uiClient, outreachDir, campaign string, hc harvestConfig) {
	p, err := withProgress(outreachDir, campaign, func(*harvestProgress) error { return nil })
	if err != nil || len(p.InFlight) == 0 {
		return
	}
	for tag, f := range p.InFlight {
		out, done := b.harvestReadJob(f.RunID)
		if !done {
			// Stale?
			if t, perr := time.Parse(time.RFC3339, f.QueuedAt); perr == nil && now.Sub(t) > harvestJobStaleAfter {
				stillPending := false
				if f.PendingPath != "" {
					if _, serr := os.Stat(f.PendingPath); serr == nil {
						stillPending = true
						_ = os.Remove(f.PendingPath) // cancel: nobody claimed it
					}
				}
				reason := "job stale after 20m"
				if stillPending {
					reason += " (never claimed — collector busy or offline)"
				}
				b.harvestFail(now, outreachDir, campaign, tag, f, reason)
			}
			continue
		}
		if strings.HasPrefix(tag, "leg:") {
			res, ierr := ingestLeg(outreachDir, campaign, f.Seed, out, hc.GoalKeywords, f.Box)
			if ierr != nil {
				log.Printf("harvest[%s/%s]: ingest leg: %v", c.Slug, campaign, ierr)
			}
			_, _ = withLedger(b.uiDataRoot, now, func(l *harvestLedger) error {
				if out.Failed {
					l.box(f.Box).recordFailure(now, out.Reason)
				} else {
					l.box(f.Box).recordSuccess()
				}
				return nil
			})
			_, _ = withProgress(outreachDir, campaign, func(p *harvestProgress) error {
				delete(p.InFlight, tag)
				return nil
			})
			_ = res
			continue
		}
		// enrich
		uid := tag
		if out.Failed && f.Attempts+1 < harvestMaxAttempts && isTransient(out.Reason) {
			// transient (timeout / no result / self-landing): retry elsewhere
			b.harvestFail(now, outreachDir, campaign, tag, f, out.Reason)
			continue
		}
		rec := map[string]any{"uid": uid, "profile_url": f.URL, "seed": f.Seed, "campaign": campaign,
			"name": f.Name, "subtitle": f.Subtitle, "attempts": f.Attempts + 1,
			"enriched_at": nowISO(), "run_id": f.RunID, "collector": f.Box}
		if !out.Failed && len(out.Items) > 0 {
			rec["record"] = out.Items[0]
			rec["ok"] = true
		} else {
			rec["ok"] = false
			rec["error"] = strOr(out.Reason, "no_record")
			if len(out.Items) > 0 {
				rec["record"] = out.Items[0]
			}
		}
		path := harvestEnrichedPath(outreachDir, campaign, uid)
		_ = os.MkdirAll(filepath.Dir(path), 0o700)
		_ = os.WriteFile(path, []byte(marshalIndentJSON(rec)+"\n"), 0o600)
		_, _ = withLedger(b.uiDataRoot, now, func(l *harvestLedger) error {
			if out.Failed {
				l.box(f.Box).recordFailure(now, out.Reason)
			} else {
				l.box(f.Box).recordSuccess()
			}
			return nil
		})
		_, _ = withProgress(outreachDir, campaign, func(p *harvestProgress) error {
			delete(p.InFlight, tag)
			p.AwaitDecision = append(p.AwaitDecision, uid)
			p.Totals["enriched"]++
			return nil
		})
		_, _ = withSeen(outreachDir, func(reg *seenRegistry) error {
			sp := reg.Profiles[uid]
			sp.Status = "enriched"
			sp.UpdatedAt = nowISO()
			reg.Profiles[uid] = sp
			return nil
		})
	}
}

// harvestFail handles a stale/transiently-failed in-flight job: ledger failure
// (may trip the breaker), then re-queue the friend or give up.
func (b *bridge) harvestFail(now time.Time, outreachDir, campaign, tag string, f harvestInFlight, reason string) {
	tripped := false
	_, _ = withLedger(b.uiDataRoot, now, func(l *harvestLedger) error {
		tripped = l.box(f.Box).recordFailure(now, reason)
		return nil
	})
	if tripped {
		log.Printf("harvest: collector %s quarantined for %s after repeated failures (%s)", f.Box, ledgerQuarantine, reason)
	}
	_, _ = withProgress(outreachDir, campaign, func(p *harvestProgress) error {
		delete(p.InFlight, tag)
		if strings.HasPrefix(tag, "leg:") {
			for i := range p.Seeds {
				if "leg:"+p.Seeds[i].UID == tag {
					p.Seeds[i].LegFailures++
					if f.Box != "" && !hasStr(p.Seeds[i].TriedBoxes, f.Box) {
						p.Seeds[i].TriedBoxes = append(p.Seeds[i].TriedBoxes, f.Box)
					}
					p.Seeds[i].LastLegAt = nowISO()
					p.Seeds[i].LastLegBox = f.Box
				}
			}
			p.Totals["leg_failures"]++
			return nil
		}
		// enrich: give the day budget back and requeue (or give up)
		if p.DayEnriched > 0 {
			p.DayEnriched--
		}
		if requeueFriend(p, f, tag, f.Box, reason) {
			// give up: park a failed record for the agent to mark enrich_failed
			rec := map[string]any{"uid": tag, "profile_url": f.URL, "seed": f.Seed, "campaign": campaign,
				"name": f.Name, "subtitle": f.Subtitle, "attempts": f.Attempts + 1,
				"enriched_at": nowISO(), "run_id": f.RunID, "collector": f.Box, "ok": false,
				"error": "gave up after " + strconvItoa(f.Attempts+1) + " attempts: " + reason}
			path := harvestEnrichedPath(outreachDir, campaign, tag)
			_ = os.MkdirAll(filepath.Dir(path), 0o700)
			_ = os.WriteFile(path, []byte(marshalIndentJSON(rec)+"\n"), 0o600)
			p.AwaitDecision = append(p.AwaitDecision, tag)
		}
		return nil
	})
}

func strconvItoa(n int) string { return strconv.Itoa(n) }

// isTransient: reasons worth a retry on another box (vs a genuine "profile has
// nothing" which the agent should judge).
func isTransient(reason string) bool {
	r := strings.ToLower(reason)
	for _, k := range []string{"timeout", "did not complete", "stale", "not claimed", "never claimed",
		"landed_on_self", "no result", "capture", "navigation", "lib not present", "no_record"} {
		if strings.Contains(r, k) {
			return true
		}
	}
	return false
}

// harvestReadJob looks for the job's output in the inbox. done=false while the
// job is running / unclaimed. On done it returns a tri-state leg outcome (see
// legOutcome): items, cursor, has_next KNOWN-or-not, and Failed+Reason when
// the collector reported an error, landed on the operator's own profile, or
// produced no records at all.
func (b *bridge) harvestReadJob(runID string) (out legOutcome, done bool) {
	// Harvest jobs land under the OWNER's tree: outputRoot/month/{owner}/harvest/{run_id}.
	matches, _ := filepath.Glob(filepath.Join(b.outputRoot, "*", "*", "harvest", safeFilename(runID)))
	if len(matches) == 0 {
		return out, false
	}
	dir := matches[0]
	status, err := readJSONFile(filepath.Join(dir, "collector_status.json"))
	if err != nil {
		return out, false
	}
	st := mStr(status, "status")
	doneFlag, _ := status["completed"].(bool)
	if !doneFlag && st != "completed" && st != "failed" && st != "error" {
		return out, false
	}
	if st == "failed" || st == "error" {
		out.Failed = true
		out.Reason = strOr(mStr(status, "message"), st)
	}
	// Source-level error string from the extension, when it wrote one.
	for _, row := range readJSONLines(filepath.Join(dir, "source_status.jsonl")) {
		if mStr(row, "status") == "error" {
			out.Failed = true
			out.Reason = strOr(mStr(row, "error"), strOr(mStr(row, "message"), "source error"))
		}
	}
	sawRecords := false
	for _, row := range readJSONLines(filepath.Join(dir, "private_data_points.jsonl")) {
		if ls, ok := row["landed_on_self"].(bool); ok && ls {
			out.Failed = true
			out.Reason = "landed_on_self"
		}
		recs := mMap(row, "records")
		if recs == nil {
			continue
		}
		sawRecords = true
		if e := mStr(recs, "error"); e != "" {
			out.Failed = true
			out.Reason = e
		}
		if fv, ok := recs["found"].(bool); ok && !fv && len(mList(recs, "items")) == 0 {
			out.Failed = true
			out.Reason = strOr(out.Reason, "no result")
		}
		for _, it := range mList(recs, "items") {
			if m, ok := it.(map[string]any); ok {
				if mStr(m, "status") == "error" {
					out.Failed = true
					out.Reason = strOr(mStr(m, "error"), "capability error")
					continue
				}
				out.Items = append(out.Items, m)
			}
		}
		if pi := mMap(recs, "page_info"); pi != nil {
			out.EndCursor = mStr(pi, "end_cursor")
			if v, ok := pi["has_next_page"].(bool); ok {
				out.HasNext, out.HasNextKnown = v, true
			}
		} else {
			if ec := mStr(recs, "end_cursor"); ec != "" {
				out.EndCursor = ec
			}
			if v, ok := recs["has_next_page"].(bool); ok {
				out.HasNext, out.HasNextKnown = v, true
			}
		}
	}
	if !sawRecords && !out.Failed {
		out.Failed = true
		out.Reason = "no_record"
	}
	return out, true
}
