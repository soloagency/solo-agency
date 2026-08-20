package main

// harvest_ledger.go — the operator-wide per-collector ledger for Leads From
// Friends. One Facebook account (one extension) serves EVERY harvest campaign
// of every client, so its daily cap, its pacing and its health cannot live in
// any single campaign's progress file — the review found that N campaigns
// each granted the same account its full cap. This file is the single truth:
//
//   {data root}/collector/harvest_ledger.json
//     day_key
//     boxes[instance_id] = { day_jobs, day_enriches, day_legs, last_job_at,
//                            consecutive_failures, quarantined_until, last_error }
//
// Circuit breaker, by KIND of failure. Counting every failure the same way and
// benching at three-in-a-row was measured wrong on 2026-08-18: the background
// rate of empty reads was ~33%, so a run of three arrives by CHANCE roughly
// every 25 attempts. The live campaign spent the day in a loop of ~20 minutes of
// work and then two hours benched, taking ~7 profiles an hour against a budget
// of 500 — and worse, a genuinely restricted account looked exactly like a noisy
// one, so the breaker had stopped carrying information.
//
//   account-side (landed_on_self, checkpoint, restricted, action_blocked):
//     ONE is enough. Waiting for three means two more actions from an account
//     Facebook has already flagged. Benched 6h; wake amnesty never lifts it.
//   soft (no_record, capability did not complete, timeout, navigation):
//     routine noise. Does NOT count toward the breaker — it SLOWS THE BOX DOWN
//     (pacingMultiplier), because the answer to being throttled is to go slower,
//     not to bench for two hours and then return at exactly the pace that earned
//     the throttling. Only a window that is almost all failures buys a short
//     cool-off, so a box that is truly getting nowhere stops burning jobs.
//   anything else: unchanged — three in a row, two hours.
//
// A success resets the counter. Read by every campaign, written under one flock, so pacing across
// campaigns is real too: last_job_at is per BOX, not per campaign.

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	ledgerFailuresToQuarantine = 3
	ledgerQuarantine           = 2 * time.Hour
	// An account-side signal is a verdict, not a flake: bench it long, and lift it
	// by a human looking at the account rather than by a timer alone.
	ledgerAccountQuarantine = 6 * time.Hour
	// A box whose recent window is almost all failures is getting nowhere; rest it
	// briefly rather than punishing it for two hours.
	ledgerCoolOff     = 30 * time.Minute
	ledgerWindow      = 20
	ledgerCoolOffRate = 0.8
	// 0% soft failures -> 1x the normal gap, ~33% -> ~2.3x, 60% -> ~3.4x, capped.
	ledgerPacingSlope = 4.0
	ledgerPacingMax   = 4.0
	ledgerPacingMinN  = 5
)

type ledgerBox struct {
	DayJobs             int    `json:"day_jobs"`
	DayEnriches         int    `json:"day_enriches"`
	DayLegs             int    `json:"day_legs"`
	LastJobAt           string `json:"last_job_at,omitempty"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	QuarantinedUntil    string `json:"quarantined_until,omitempty"`
	LastError           string `json:"last_error,omitempty"`
	TotalJobs           int    `json:"total_jobs"`
	TotalFailures       int    `json:"total_failures"`
	// Recent: the last ledgerWindow outcomes for this box, newest last, '1' ok and
	// '0' a soft failure. This is what turns "we are being throttled" from a guess
	// into a number the pacing can read.
	Recent string `json:"recent_outcomes,omitempty"`
}

// isAccountSideFailure: symptoms that say THIS ACCOUNT is in trouble, as opposed
// to a page that did not render. One of these is worth more than three of the
// other kind, and no amount of waiting makes it a machine problem.
func isAccountSideFailure(reason string) bool {
	r := strings.ToLower(reason)
	for _, k := range []string{"landed_on_self", "checkpoint", "restricted", "action_blocked",
		"temporarily blocked", "community standards", "not_a_member"} {
		if strings.Contains(r, k) {
			return true
		}
	}
	return false
}

// isSoftFailure: the collector reached Facebook and came back with nothing
// usable. Common, expected, and NOT evidence that the account is in trouble.
func isSoftFailure(reason string) bool {
	if isAccountSideFailure(reason) {
		return false
	}
	r := strings.ToLower(reason)
	for _, k := range []string{"no_record", "no result", "did not complete", "timeout",
		"capture", "navigation", "stale", "never claimed", "source error", "returned nothing"} {
		if strings.Contains(r, k) {
			return true
		}
	}
	return false
}

func (bx *ledgerBox) pushOutcome(c byte) {
	bx.Recent += string(c)
	if len(bx.Recent) > ledgerWindow {
		bx.Recent = bx.Recent[len(bx.Recent)-ledgerWindow:]
	}
}

// softFailRate over the recent window, plus how many outcomes it is based on.
func (bx *ledgerBox) softFailRate() (float64, int) {
	n := len(bx.Recent)
	if n == 0 {
		return 0, 0
	}
	bad := 0
	for i := 0; i < n; i++ {
		if bx.Recent[i] == '0' {
			bad++
		}
	}
	return float64(bad) / float64(n), n
}

// pacingMultiplier stretches the gap between this box's jobs as its recent
// failure rate rises. This is the whole point of the redesign: being throttled
// is answered by going slower, continuously, instead of by a two-hour bench
// followed by a return to the pace that caused it.
func (bx *ledgerBox) pacingMultiplier() float64 {
	if bx == nil {
		return 1
	}
	rate, n := bx.softFailRate()
	if n < ledgerPacingMinN {
		return 1
	}
	m := 1 + rate*ledgerPacingSlope
	if m > ledgerPacingMax {
		m = ledgerPacingMax
	}
	return m
}

type harvestLedger struct {
	SchemaVersion int                   `json:"schema_version"`
	DayKey        string                `json:"day_key"`
	Boxes         map[string]*ledgerBox `json:"boxes"`
	UpdatedAt     string                `json:"updated_at"`
}

func harvestLedgerPath(dataRoot string) string {
	return filepath.Join(dataRoot, "collector", "harvest_ledger.json")
}

func withLedger(dataRoot string, now time.Time, fn func(*harvestLedger) error) (*harvestLedger, error) {
	l := &harvestLedger{}
	err := withLockedJSON(harvestLedgerPath(dataRoot),
		func(raw []byte) error {
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, l); err != nil {
					return fmt.Errorf("harvest_ledger.json is corrupt: %w", err)
				}
			}
			if l.SchemaVersion == 0 {
				l.SchemaVersion = 1
			}
			if l.Boxes == nil {
				l.Boxes = map[string]*ledgerBox{}
			}
			if key := now.Format("2006-01-02"); l.DayKey != key {
				l.DayKey = key
				for _, bx := range l.Boxes {
					bx.DayJobs, bx.DayEnriches, bx.DayLegs = 0, 0, 0
				}
			}
			return nil
		},
		func() ([]byte, error) { return json.MarshalIndent(l, "", "  ") },
		func() error {
			if err := fn(l); err != nil {
				return err
			}
			l.UpdatedAt = now.UTC().Format(time.RFC3339)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (l *harvestLedger) box(id string) *ledgerBox {
	if l.Boxes[id] == nil {
		l.Boxes[id] = &ledgerBox{}
	}
	return l.Boxes[id]
}

// quarantined reports whether the box is currently benched.
func (bx *ledgerBox) quarantined(now time.Time) bool {
	if bx == nil || bx.QuarantinedUntil == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, bx.QuarantinedUntil)
	return err == nil && now.Before(t)
}

// eligible: not quarantined, under the per-box daily cap, and past the
// per-box pacing gap (the gap is measured on the BOX, across campaigns).
func (bx *ledgerBox) eligible(now time.Time, cap int, gap time.Duration) bool {
	if bx == nil {
		return true
	}
	if bx.quarantined(now) || bx.DayJobs >= cap {
		return false
	}
	if bx.LastJobAt != "" {
		want := time.Duration(float64(gap) * bx.pacingMultiplier())
		if t, err := time.Parse(time.RFC3339, bx.LastJobAt); err == nil && now.Sub(t) < want {
			return false
		}
	}
	return true
}

func (bx *ledgerBox) recordJob(now time.Time, kind string) {
	bx.DayJobs++
	bx.TotalJobs++
	if kind == "leg" {
		bx.DayLegs++
	} else {
		bx.DayEnriches++
	}
	bx.LastJobAt = now.UTC().Format(time.RFC3339)
}

func (bx *ledgerBox) recordSuccess() {
	bx.ConsecutiveFailures = 0
	bx.QuarantinedUntil = ""
	bx.LastError = ""
	bx.pushOutcome('1')
}

// recordFailure reacts to a failure according to what KIND it is (see the file
// header). Returns true when the box just entered quarantine.
func (bx *ledgerBox) recordFailure(now time.Time, reason string) bool {
	bx.TotalFailures++
	bx.LastError = reason
	bx.pushOutcome('0')
	bench := func(d time.Duration) bool {
		if bx.quarantined(now) {
			return false
		}
		bx.QuarantinedUntil = now.Add(d).UTC().Format(time.RFC3339)
		return true
	}
	switch {
	case isAccountSideFailure(reason):
		// Stamp the counter at the threshold so every surface still reads "tripped".
		bx.ConsecutiveFailures = ledgerFailuresToQuarantine
		return bench(ledgerAccountQuarantine)
	case isSoftFailure(reason):
		// Noise: no breaker, no counter. The pacing multiplier already slowed this
		// box down. Only an almost-entirely-failing window earns a short rest.
		if rate, n := bx.softFailRate(); n >= ledgerWindow && rate >= ledgerCoolOffRate {
			return bench(ledgerCoolOff)
		}
		return false
	default:
		bx.ConsecutiveFailures++
		if bx.ConsecutiveFailures >= ledgerFailuresToQuarantine {
			return bench(ledgerQuarantine)
		}
		return false
	}
}
