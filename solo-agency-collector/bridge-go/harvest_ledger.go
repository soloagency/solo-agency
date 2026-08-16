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
// Circuit breaker: 3 consecutive failures (timeout / stale / landed_on_self /
// error record) put the box in quarantine for 2h — the daemon simply stops
// picking it, other boxes carry on (that IS the failover), and after the
// quarantine one probe job decides whether it is back. A success resets the
// counter. Read by every campaign, written under one flock, so pacing across
// campaigns is real too: last_job_at is per BOX, not per campaign.

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
)

const (
	ledgerFailuresToQuarantine = 3
	ledgerQuarantine           = 2 * time.Hour
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
		if t, err := time.Parse(time.RFC3339, bx.LastJobAt); err == nil && now.Sub(t) < gap {
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
}

// recordFailure counts a failure and trips the breaker at the threshold.
// Returns true when the box just entered quarantine.
func (bx *ledgerBox) recordFailure(now time.Time, reason string) bool {
	bx.ConsecutiveFailures++
	bx.TotalFailures++
	bx.LastError = reason
	if bx.ConsecutiveFailures >= ledgerFailuresToQuarantine && !bx.quarantined(now) {
		bx.QuarantinedUntil = now.Add(ledgerQuarantine).UTC().Format(time.RFC3339)
		return true
	}
	return false
}
