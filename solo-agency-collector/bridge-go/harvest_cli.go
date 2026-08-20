package main

// harvest_cli.go — `tool crm-store harvest <op>`: the agent's window into a
// Leads-From-Friends campaign. The daemon (harvest_daemon.go) does the
// mechanical walking; these ops are what a run calls to SEED a campaign, to
// take a batch of enriched friends for judgement, and to record verdicts.
//
//   harvest status  --campaign X
//   harvest seed    --campaign X            (re)sync seed_profiles from the campaign config
//   harvest pending --campaign X [--limit N] → enriched records awaiting a keep/reject
//   harvest decide  --campaign X --profile URL --status kept|rejected|enrich_failed [--lead-id ID] [--reason R]
//   harvest ingest  --campaign X --seed URL --records F [--end-cursor C] [--has-next true|false] [--box B]
//                                                     (daemon-internal, exposed for tests/manual repair)

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runHarvestCLI(store *crmStore, a *cliArgs, op string) int {
	campaign := a.get("--campaign")
	if campaign == "" && op != "" {
		return crmUsageErr("harvest needs --campaign")
	}
	clientDir := store.clientDir
	cfg := store.getCampaign(campaign)
	if cfg == nil {
		return crmFail(fmt.Errorf("campaign %q not found", campaign))
	}
	ch := mStr(cfg, "channel_strategy")
	if ch != harvestChannel && ch != zillowChannel {
		return crmFail(fmt.Errorf("campaign %q is channel %q, not %s or %s", campaign, ch, harvestChannel, zillowChannel))
	}
	settings := loadSystemSettings(pipelineRootFromClientDir(clientDir))
	hc := harvestConfigFrom(cfg, settings)
	hc.Channel = ch
	hc.Zillow = zillowConfigFrom(cfg)

	switch op {
	case "seed":
		if hc.Channel == zillowChannel {
			return crmOut(map[string]any{"ok": true, "note": "zillow campaigns have no seeds — the daemon walks zillow_locations × zillow_keywords",
				"locations": hc.Zillow.Locations, "keywords": hc.Zillow.Keywords}, 0)
		}
		p, err := syncSeeds(clientDir, campaign, hc)
		if err != nil {
			return crmFail(err)
		}
		return crmOut(map[string]any{"ok": true, "seeds": p.Seeds, "current_seed": p.CurrentSeed}, 0)

	case "status":
		p, err := withProgress(clientDir, campaign, func(*harvestProgress) error { return nil })
		if err != nil {
			return crmFail(err)
		}
		out := map[string]any{"ok": true, "campaign": campaign, "channel": hc.Channel, "config": map[string]any{
			"daily_budget": hc.DailyBudget, "per_collector_budget": hc.PerBoxBudget,
			"leg_pages": hc.LegPages, "quiet_from": hc.QuietFrom, "quiet_to": hc.QuietTo,
			"seed_profiles": hc.SeedProfiles, "goal_keywords": hc.GoalKeywords,
			"zillow_locations": hc.Zillow.Locations, "zillow_keywords": hc.Zillow.Keywords},
			"seeds": p.Seeds, "current_seed": p.CurrentSeed, "queue_len": len(p.Queue),
			"in_flight": len(p.InFlight), "await_decision": len(p.AwaitDecision),
			"day_key": p.DayKey, "day_enriched": p.DayEnriched, "day_per_box": p.DayPerBox,
			"totals": p.Totals, "updated_at": p.UpdatedAt}
		if hc.Channel == zillowChannel {
			out["zillow"] = p.zillow() // always present, even before the first tick
		}
		return crmOut(out, 0)

	case "pending":
		limit := a.getInt("--limit", harvestDecideBatch)
		p, err := withProgress(clientDir, campaign, func(*harvestProgress) error { return nil })
		if err != nil {
			return crmFail(err)
		}
		var out []map[string]any
		for _, uid := range p.AwaitDecision {
			if len(out) >= limit {
				break
			}
			rec, rerr := readJSONFile(harvestEnrichedPath(clientDir, campaign, uid))
			if rerr != nil {
				continue
			}
			out = append(out, rec)
		}
		return crmOut(map[string]any{"ok": true, "campaign": campaign, "goal_description": hc.GoalDescription,
			"goal_keywords": hc.GoalKeywords, "count": len(out), "remaining": len(p.AwaitDecision),
			"records": out}, 0)

	case "decide":
		profile := a.get("--profile")
		uid, _ := sourceUID(profile)
		if uid == "" {
			return crmUsageErr("decide needs --profile (a profile URL)")
		}
		// Refuse a decision for a uid nobody is waiting on — a differently
		// canonicalised url would otherwise leave the real record pending forever.
		p, perr := withProgress(clientDir, campaign, func(*harvestProgress) error { return nil })
		if perr != nil {
			return crmFail(perr)
		}
		if !hasStr(p.AwaitDecision, uid) {
			return crmFail(fmt.Errorf("uid %q is not awaiting a decision in campaign %q — pass the pending record's profile_url exactly (%d pending)", uid, campaign, len(p.AwaitDecision)))
		}
		if err := recordDecision(clientDir, campaign, uid, a.get("--status"), a.get("--lead-id"), a.get("--reason")); err != nil {
			return crmFail(err)
		}
		return crmOut(map[string]any{"ok": true, "uid": uid, "status": a.get("--status"), "lead_id": a.get("--lead-id")}, 0)

	case "ingest":
		seed := a.get("--seed")
		f := a.get("--records")
		if seed == "" || f == "" {
			return crmUsageErr("ingest needs --seed and --records (JSON array of ProfileSummary)")
		}
		raw, err := os.ReadFile(f)
		if err != nil {
			return crmFail(err)
		}
		var items []map[string]any
		if err := json.Unmarshal(raw, &items); err != nil {
			return crmFail(fmt.Errorf("records must be a JSON array: %w", err))
		}
		out := legOutcome{Items: items, EndCursor: a.get("--end-cursor")}
		if hn := strings.ToLower(a.get("--has-next")); hn == "true" || hn == "false" {
			out.HasNextKnown = true
			out.HasNext = hn == "true"
		}
		res, err := ingestLeg(clientDir, campaign, seed, out, hc.GoalKeywords, a.get("--box"))
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	}
	return crmUsageErr("harvest op must be status | seed | pending | decide | ingest")
}

// pipelineRootFromClientDir: clients/{slug}/{biz_loc}/outreach → data root.
func pipelineRootFromClientDir(clientDir string) string {
	d := clientDir
	for i := 0; i < 6; i++ {
		if filepath.Base(d) == "clients" {
			return filepath.Dir(d)
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return clientDir
}

// syncSeeds mirrors the campaign's seed_profiles into progress.json (adding
// new seeds, never dropping cursors of existing ones) and picks current_seed.
func syncSeeds(clientDir, campaign string, hc harvestConfig) (*harvestProgress, error) {
	if len(hc.SeedProfiles) == 0 {
		return nil, fmt.Errorf("campaign has no seed_profiles — add profile URLs (one per line in the UI)")
	}
	return withProgress(clientDir, campaign, func(p *harvestProgress) error {
		have := map[string]bool{}
		want := map[string]bool{}
		for _, raw := range hc.SeedProfiles {
			if clean, ok := canonicalStoreURL(raw); ok {
				if uid, _ := sourceUID(clean); uid != "" {
					want[uid] = true
				}
			}
		}
		for i := range p.Seeds {
			have[p.Seeds[i].UID] = true
			// An operator running `harvest seed` is saying "try these again". It is the
			// only documented way back from a seed error, which is otherwise a one-way
			// door: nothing else clears Error, and Error is what bars the seed from
			// selection. The cursor is kept, so a partly-walked seed resumes rather
			// than restarting.
			if want[p.Seeds[i].UID] && p.Seeds[i].Error != "" {
				p.Seeds[i].Error = ""
				p.Seeds[i].LegFailures = 0
				p.Seeds[i].TriedBoxes = nil
			}
		}
		// Let the daemon re-pick: the stored pointer may name a seed that is now
		// removed, or the one that was stuck.
		p.CurrentSeed = ""
		for _, raw := range hc.SeedProfiles {
			clean, ok := canonicalStoreURL(raw)
			if !ok {
				continue
			}
			uid, _ := sourceUID(clean)
			if uid == "" || have[uid] {
				continue
			}
			fu := friendsURL(clean)
			if fu == "" {
				continue
			}
			p.Seeds = append(p.Seeds, harvestSeed{URL: clean, UID: uid, FriendsURL: fu})
			have[uid] = true
		}
		// The seed profiles themselves are "seen" — never enrich a seed as if it were a friend.
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
			// Same eligibility test the daemon uses (harvest_daemon.go reconcileSeeds).
			// It used to accept any un-exhausted seed, so a re-sync could pin the pointer
			// to a removed or errored seed, and the daemon only re-picks when the pointer
			// is empty: the campaign would sit on a seed it is not allowed to walk.
			for _, s := range p.Seeds {
				if !s.Exhausted && !s.Removed && s.Error == "" {
					p.CurrentSeed = s.URL
					break
				}
			}
		}
		return nil
	})
}
