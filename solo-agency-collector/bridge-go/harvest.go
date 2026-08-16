package main

// harvest.go — "Leads From Friends": a campaign channel that walks the friend
// lists of operator-chosen seed profiles, enriches each friend once, and lets
// the agent keep the ones that match the campaign goal straight into the CRM.
//
// Why this shape:
//   - The friend list of a well-connected professional is the highest-quality
//     lead source this system has (people who already trust the seed). It is
//     also the most account-sensitive: reading 500-1000 profiles a day is a
//     footprint an order of magnitude beyond anything else here. So pacing,
//     collector rotation, per-collector caps and a nightly pause all live in
//     CODE, in this file, not in prose an agent may or may not honour.
//   - Seeds share many mutual friends. A CLIENT-WIDE `seen_profiles.json`
//     (keyed by sourceUID) makes "already handled" one lookup, across all
//     harvest campaigns of the client, forever.
//   - The friend list is read in LEGS (fb.profile.friends start_cursor /
//     end_cursor, ~336 rows per leg); the cursor lives in progress.json per
//     seed, so a list of thousands drains across days and restarts.
//   - The daemon (harvest_daemon.go) does the mechanical loop — enqueue leg,
//     ingest, enqueue enrich, sleep 20-40s, rotate collector, respect budgets.
//     The AGENT does the judgement: `harvest pending` hands it a small batch of
//     enriched records, `harvest decide` records keep/reject per friend, and a
//     kept friend becomes a CRM contact immediately (nothing is sent anywhere,
//     so there is no approval gate — the operator ruled that explicitly).
//
// State (all under {client}/outreach/harvest/):
//   seen_profiles.json            — client-wide: uid -> {status, seed, lead_id, ts}
//   {campaign}/progress.json      — per seed cursor/legs/counters + queues + budgets
//   {campaign}/enriched/{uid}.json — the enrich record waiting for a decision

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	harvestChannel          = "friend_harvest"
	harvestDefaultDaily     = 500
	harvestDefaultPerBox    = 150
	harvestDefaultLegPages  = 10
	harvestMinGapSec        = 20
	harvestMaxGapSec        = 40
	harvestDefaultQuietFrom = "01:00"
	harvestDefaultQuietTo   = "06:00"
	harvestDecideBatch      = 25
)

// --- seen profiles (client-wide) --------------------------------------------------

type seenProfile struct {
	UID       string `json:"uid"`
	URL       string `json:"url"`
	Name      string `json:"name,omitempty"`
	Status    string `json:"status"` // queued | enriched | kept | rejected | enrich_failed | seed
	Seed      string `json:"seed,omitempty"`
	Campaign  string `json:"campaign,omitempty"`
	LeadID    string `json:"lead_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
	FirstSeen string `json:"first_seen"`
	UpdatedAt string `json:"updated_at"`
}

type seenRegistry struct {
	SchemaVersion int                    `json:"schema_version"`
	Profiles      map[string]seenProfile `json:"profiles"`
}

func harvestDir(clientDir string) string { return filepath.Join(clientDir, "harvest") }
func seenProfilesPath(clientDir string) string {
	return filepath.Join(harvestDir(clientDir), "seen_profiles.json")
}

func withSeen(clientDir string, fn func(*seenRegistry) error) (*seenRegistry, error) {
	reg := &seenRegistry{}
	err := withLockedJSON(seenProfilesPath(clientDir),
		func(raw []byte) error {
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, reg); err != nil {
					return fmt.Errorf("seen_profiles.json is corrupt: %w", err)
				}
			}
			if reg.SchemaVersion == 0 {
				reg.SchemaVersion = 1
			}
			if reg.Profiles == nil {
				reg.Profiles = map[string]seenProfile{}
			}
			return nil
		},
		func() ([]byte, error) { return json.MarshalIndent(reg, "", "  ") },
		func() error { return fn(reg) })
	if err != nil {
		return nil, err
	}
	return reg, nil
}

// --- per-campaign progress ------------------------------------------------------

type harvestSeed struct {
	URL         string `json:"url"`
	UID         string `json:"uid"`
	FriendsURL  string `json:"friends_url"`
	EndCursor   string `json:"end_cursor,omitempty"`
	LegsDone    int    `json:"legs_done"`
	FriendsSeen int    `json:"friends_seen"`
	Exhausted   bool   `json:"exhausted"`
	LastLegAt   string `json:"last_leg_at,omitempty"`
	LastLegBox  string `json:"last_leg_box,omitempty"`
	LegFailures int    `json:"leg_failures"`           // consecutive failed/empty legs; reset on a real leg
	TriedBoxes  []string `json:"tried_boxes,omitempty"` // boxes that returned NOTHING for this seed (visibility)
	Removed     bool   `json:"removed,omitempty"`      // seed dropped from the campaign config; cursor kept for re-add
	Error       string `json:"error,omitempty"`
}

type harvestQueued struct {
	UID      string `json:"uid"`
	URL      string `json:"url"`
	Name     string `json:"name,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
	Seed     string `json:"seed"`
	Priority int    `json:"priority"` // 0 = subtitle matched goal keywords, 1 = otherwise
	QueuedAt string `json:"queued_at"`
	Attempts int    `json:"attempts,omitempty"`   // enrich attempts so far (requeued after stale/transient failure)
	AvoidBox string `json:"avoid_box,omitempty"`  // do not hand the retry to the box that just failed it
}

type harvestProgress struct {
	SchemaVersion int                        `json:"schema_version"`
	Campaign      string                     `json:"campaign"`
	Seeds         []harvestSeed              `json:"seeds"`
	CurrentSeed   string                     `json:"current_seed"`
	Queue         []harvestQueued            `json:"queue"`         // awaiting enrich
	InFlight      map[string]harvestInFlight `json:"in_flight"`     // uid -> enrich job
	AwaitDecision []string                   `json:"await_decision"` // uids with enriched/{uid}.json
	DayKey        string                     `json:"day_key"`
	DayEnriched   int                        `json:"day_enriched"`
	DayPerBox     map[string]int             `json:"day_per_box"`
	LastEnrichAt  string                     `json:"last_enrich_at,omitempty"`
	LastBox       string                     `json:"last_box,omitempty"`
	Totals        map[string]int             `json:"totals"`
	Zillow        *zillowCursor              `json:"zillow,omitempty"` // Leads From Zillow cursor (harvest_zillow.go)
	UpdatedAt     string                     `json:"updated_at"`
}

type harvestInFlight struct {
	URL         string `json:"url"`
	Seed        string `json:"seed"`
	RunID       string `json:"run_id"`
	Box         string `json:"box"`
	QueuedAt    string `json:"queued_at"`
	Name        string `json:"name,omitempty"`
	Subtitle    string `json:"subtitle,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Attempts    int    `json:"attempts,omitempty"`
	PendingPath string `json:"pending_path,omitempty"` // jobs/pending file; removable while unclaimed
}

func harvestCampaignDir(clientDir, campaign string) string {
	return filepath.Join(harvestDir(clientDir), safeFilename(campaign))
}
func harvestProgressPath(clientDir, campaign string) string {
	return filepath.Join(harvestCampaignDir(clientDir, campaign), "progress.json")
}
func harvestEnrichedPath(clientDir, campaign, uid string) string {
	return filepath.Join(harvestCampaignDir(clientDir, campaign), "enriched", uidHash(uid)+".json")
}

func withProgress(clientDir, campaign string, fn func(*harvestProgress) error) (*harvestProgress, error) {
	p := &harvestProgress{}
	err := withLockedJSON(harvestProgressPath(clientDir, campaign),
		func(raw []byte) error {
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, p); err != nil {
					return fmt.Errorf("progress.json is corrupt: %w", err)
				}
			}
			if p.SchemaVersion == 0 {
				p.SchemaVersion = 1
			}
			p.Campaign = campaign
			if p.InFlight == nil {
				p.InFlight = map[string]harvestInFlight{}
			}
			if p.DayPerBox == nil {
				p.DayPerBox = map[string]int{}
			}
			if p.Totals == nil {
				p.Totals = map[string]int{}
			}
			return nil
		},
		func() ([]byte, error) { return json.MarshalIndent(p, "", "  ") },
		func() error {
			if err := fn(p); err != nil {
				return err
			}
			p.UpdatedAt = nowISO()
			return nil
		})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// --- campaign config for the channel ---------------------------------------------

type harvestConfig struct {
	Channel        string       // friend_harvest | zillow_harvest
	Zillow         zillowConfig // Leads From Zillow inputs (harvest_zillow.go)
	SeedProfiles   []string
	DailyBudget    int
	PerBoxBudget   int
	LegPages       int
	QuietFrom      string
	QuietTo        string
	GoalKeywords   []string
	GoalDescription string
}

// harvestConfigFrom reads the campaign document. Budgets default from the
// operator's global system settings when the campaign leaves them blank.
func harvestConfigFrom(cfg map[string]any, settings systemSettings) harvestConfig {
	h := mMap(cfg, "harvest")
	out := harvestConfig{
		DailyBudget:  mInt(h, "daily_budget", 0),
		PerBoxBudget: mInt(h, "per_collector_budget", 0),
		LegPages:     mInt(h, "leg_pages", harvestDefaultLegPages),
		QuietFrom:    mStr(h, "quiet_from"),
		QuietTo:      mStr(h, "quiet_to"),
	}
	if out.DailyBudget <= 0 {
		out.DailyBudget = settings.HarvestDailyBudget
	}
	if out.PerBoxBudget <= 0 {
		out.PerBoxBudget = settings.HarvestPerCollectorBudget
	}
	if out.LegPages <= 0 || out.LegPages > 40 {
		out.LegPages = harvestDefaultLegPages
	}
	if out.QuietFrom == "" {
		out.QuietFrom = settings.HarvestQuietFrom
	}
	if out.QuietTo == "" {
		out.QuietTo = settings.HarvestQuietTo
	}
	for _, s := range mList(cfg, "seed_profiles") {
		if u := strings.TrimSpace(sprint(s)); u != "" {
			out.SeedProfiles = append(out.SeedProfiles, u)
		}
	}
	goal := mMap(cfg, "goal")
	out.GoalDescription = mStr(goal, "description")
	for _, k := range mList(h, "goal_keywords") {
		if s := strings.ToLower(strings.TrimSpace(sprint(k))); s != "" {
			out.GoalKeywords = append(out.GoalKeywords, s)
		}
	}
	return out
}

// friendsURL builds the friend-list URL the capability's url_match expects.
func friendsURL(profileURL string) string {
	clean, ok := canonicalStoreURL(profileURL)
	if !ok {
		return ""
	}
	if strings.Contains(clean, "profile.php?id=") {
		return clean + "&sk=friends"
	}
	return strings.TrimRight(clean, "/") + "/friends"
}

// inQuietHours reports whether local time t falls inside [from, to) — a
// window that may cross midnight ("23:00"-"06:00").
func inQuietHours(t time.Time, from, to string) bool {
	parse := func(s string) (int, bool) {
		if !runTimeRe.MatchString(s) {
			return 0, false
		}
		var h, m int
		fmt.Sscanf(s, "%d:%d", &h, &m)
		return h*60 + m, true
	}
	f, ok1 := parse(from)
	e, ok2 := parse(to)
	if !ok1 || !ok2 || f == e {
		return false
	}
	cur := t.Hour()*60 + t.Minute()
	if f < e {
		return cur >= f && cur < e
	}
	return cur >= f || cur < e
}

// randomGap is the pause between two enrich jobs: uniform 20-40s. Lives in code
// so no agent ever "optimises" it away.
func randomGap(r *rand.Rand) time.Duration {
	return time.Duration(harvestMinGapSec+r.Intn(harvestMaxGapSec-harvestMinGapSec+1)) * time.Second
}

// subtitleMatches is the cheap pre-filter the capability doc recommends: a
// friend whose list subtitle already names a goal keyword is enriched first.
func subtitleMatches(subtitle string, keywords []string) bool {
	if subtitle == "" || len(keywords) == 0 {
		return false
	}
	s := strings.ToLower(subtitle)
	for _, k := range keywords {
		if k != "" && strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// resetDayIfNeeded rolls the daily counters when the local day changes.
func (p *harvestProgress) resetDayIfNeeded(now time.Time) {
	key := now.Format("2006-01-02")
	if p.DayKey != key {
		p.DayKey = key
		p.DayEnriched = 0
		p.DayPerBox = map[string]int{}
	}
}

// --- ingest a friends leg ---------------------------------------------------------

type ingestResult struct {
	Seed        string `json:"seed"`
	Received    int    `json:"received"`
	NewQueued   int    `json:"new_queued"`
	AlreadySeen int    `json:"already_seen"`
	Prioritized int    `json:"prioritized"`
	EndCursor   string `json:"end_cursor,omitempty"`
	Exhausted   bool   `json:"exhausted"`
	LegFailed   bool   `json:"leg_failed,omitempty"`
	SeedError   string `json:"seed_error,omitempty"`
}

// legOutcome is what the daemon learned from one friends leg. hasNextKnown
// distinguishes "the connection reported its end" from "we got no signal at
// all" (records null, error item, pagination skipped, tab error) — only the
// former may exhaust a seed. The review found the old bool collapsed both.
type legOutcome struct {
	Items        []map[string]any
	EndCursor    string
	HasNext      bool
	HasNextKnown bool
	Failed       bool   // no usable records / error item / landed_on_self
	Reason       string // collector's own error string when it gave one
	Zillow       zillowLegFacts // directory envelope facts for zillow.agents.list
}

// terminalSeen: statuses that mean "this person is handled" for the whole
// client. `queued` is NOT terminal — a friend parked in some campaign's queue
// may be adopted when that campaign is gone (review finding).
func terminalSeen(status string) bool {
	switch status {
	case "enriched", "kept", "rejected", "enrich_failed", "seed":
		return true
	}
	return false
}

// ingestLeg folds one friends leg into the campaign. Progress (queue + cursor)
// is written FIRST and the seen registry second: if the second write fails,
// the worst case is a friend queued twice (benign) — never a friend marked
// seen but queued nowhere (the old order's failure).
func ingestLeg(clientDir, campaign, seedURL string, out legOutcome, keywords []string, box string) (ingestResult, error) {
	res := ingestResult{Seed: seedURL, Received: len(out.Items), EndCursor: out.EndCursor, LegFailed: out.Failed}
	now := nowISO()

	// Read seen (no write) to compute the fresh set.
	seenReg, err := withSeen(clientDir, func(*seenRegistry) error { return nil })
	if err != nil {
		return res, err
	}
	var fresh []harvestQueued
	for _, it := range out.Items {
		u := mStr(it, "url")
		uid, _ := sourceUID(u)
		if uid == "" {
			continue
		}
		if sp, dup := seenReg.Profiles[uid]; dup && (terminalSeen(sp.Status) || sp.Campaign == campaign) {
			res.AlreadySeen++
			continue
		}
		clean, _ := canonicalStoreURL(u)
		q := harvestQueued{UID: uid, URL: clean, Name: mStr(it, "name"), Subtitle: mStr(it, "subtitle"),
			Seed: seedURL, Priority: 1, QueuedAt: now}
		if subtitleMatches(q.Subtitle, keywords) {
			q.Priority = 0
			res.Prioritized++
		}
		fresh = append(fresh, q)
	}
	res.NewQueued = len(fresh)

	_, err = withProgress(clientDir, campaign, func(p *harvestProgress) error {
		p.Queue = append(p.Queue, fresh...)
		sort.SliceStable(p.Queue, func(i, j int) bool { return p.Queue[i].Priority < p.Queue[j].Priority })
		for i := range p.Seeds {
			s := &p.Seeds[i]
			if s.URL != seedURL {
				continue
			}
			s.LastLegAt = now
			s.LastLegBox = box
			if out.Failed || (!out.HasNextKnown && len(out.Items) == 0) {
				// No signal: keep the cursor, do not exhaust, count the failure.
				s.LegFailures++
				if box != "" && !hasStr(s.TriedBoxes, box) {
					s.TriedBoxes = append(s.TriedBoxes, box)
				}
				if s.LegsDone == 0 && s.LegFailures >= 2 {
					s.Error = "friend list returned nothing on " + fmt.Sprint(len(s.TriedBoxes)) +
						" collector(s) — private to those accounts, or the profile could not be read; " +
						"an account that is friends with the seed may be needed"
					res.SeedError = s.Error
				} else if s.LegFailures >= 4 {
					s.Error = "4 consecutive leg failures: " + out.Reason
					res.SeedError = s.Error
				}
				p.Totals["leg_failures"]++
				return nil
			}
			// A real leg.
			s.LegFailures = 0
			s.LegsDone++
			s.FriendsSeen += len(out.Items)
			if out.EndCursor != "" {
				s.EndCursor = out.EndCursor
			}
			s.Exhausted = out.HasNextKnown && !out.HasNext
			res.Exhausted = s.Exhausted
		}
		p.Totals["legs"]++
		p.Totals["friends_seen"] += len(out.Items)
		p.Totals["already_known"] += res.AlreadySeen
		return nil
	})
	if err != nil {
		return res, err
	}
	if len(fresh) == 0 {
		return res, nil
	}
	_, err = withSeen(clientDir, func(reg *seenRegistry) error {
		for _, q := range fresh {
			reg.Profiles[q.UID] = seenProfile{UID: q.UID, URL: q.URL, Name: q.Name, Status: "queued",
				Seed: seedURL, Campaign: campaign, FirstSeen: now, UpdatedAt: now}
		}
		return nil
	})
	return res, err
}

func hasStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// requeueFriend puts a popped friend back at the FRONT of the queue after a
// stale/transient failure, remembering the box to avoid. Past maxAttempts it
// becomes enrich_failed for real (3 different boxes tried is enough evidence).
const harvestMaxAttempts = 3

func requeueFriend(p *harvestProgress, f harvestInFlight, uid, box, reason string) (gaveUp bool) {
	if f.Attempts+1 >= harvestMaxAttempts {
		p.Totals["enrich_gave_up"]++
		return true
	}
	q := harvestQueued{UID: uid, URL: f.URL, Name: f.Name, Subtitle: f.Subtitle, Seed: f.Seed,
		Priority: 0, QueuedAt: nowISO(), Attempts: f.Attempts + 1, AvoidBox: box}
	p.Queue = append([]harvestQueued{q}, p.Queue...)
	p.Totals["requeued"]++
	return false
}

// --- decisions --------------------------------------------------------------------

// recordDecision applies the agent's verdict for one enriched friend. `kept`
// requires a lead_id (the CRM contact the caller created or matched).
func recordDecision(clientDir, campaign, uid, status, leadID, reason string) error {
	if status != "kept" && status != "rejected" && status != "enrich_failed" {
		return fmt.Errorf("status must be kept, rejected or enrich_failed")
	}
	if status == "kept" && leadID == "" {
		return fmt.Errorf("kept requires --lead-id (the CRM contact created or matched)")
	}
	now := nowISO()
	_, err := withSeen(clientDir, func(reg *seenRegistry) error {
		sp, ok := reg.Profiles[uid]
		if !ok {
			sp = seenProfile{UID: uid, FirstSeen: now, Campaign: campaign}
		}
		sp.Status, sp.LeadID, sp.Reason, sp.UpdatedAt = status, leadID, reason, now
		reg.Profiles[uid] = sp
		return nil
	})
	if err != nil {
		return err
	}
	_, err = withProgress(clientDir, campaign, func(p *harvestProgress) error {
		kept := p.AwaitDecision[:0]
		for _, u := range p.AwaitDecision {
			if u != uid {
				kept = append(kept, u)
			}
		}
		p.AwaitDecision = kept
		delete(p.InFlight, uid)
		p.Totals[status]++
		return nil
	})
	if err != nil {
		return err
	}
	_ = os.Remove(harvestEnrichedPath(clientDir, campaign, uid))
	return nil
}
