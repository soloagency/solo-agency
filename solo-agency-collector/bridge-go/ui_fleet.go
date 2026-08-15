package main

// ui_fleet.go — the operator's fleet dashboard (/ui/fleet): every client's
// posting + performance state on one screen, worst first. The screen only
// READS fleet/{client_slug}.json snapshots that each client's run rewrites at
// its end (04 step 33c) — nothing here queries a provider, ever, so the page
// is instant regardless of fleet size.
//
// Colors follow the accountability ladder: yellow when a client is
// APPROACHING the posting-gap threshold (>= 2/3 of it) or already breached
// with reminders 1-2; red from the 3rd reminder / operator escalation; gray
// when the run had no posting data at all. A "stale" marker means the
// snapshot itself is old for that client's cadence — the automation may be
// stuck, which no other screen surfaces.

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type fleetRow struct {
	Client     string
	Name       string
	Color      string // ok | warn | err | none (css dot class suffix)
	Rank       int    // 0 red, 1 yellow, 2 gray/no-data, 3 green — sort key
	LastPost   string
	GapHours   float64
	Reminders  string
	Posts      string
	Views7     string
	Likes7     string
	Leads      string
	IdeasQ     string
	DraftsPend string
	ReportHref string
	Updated    string
	Stale      bool
}

func fleetNum(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return "–"
	}
	if f, ok := v.(float64); ok {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%v", v)
}

// fleetRows loads every fleet snapshot and computes display + ranking. The
// snapshot's own posting.status (written by the run, which knows the
// per-client threshold override) wins; the global threshold only backfills
// snapshots that carry a gap without a status.
func fleetRows(dataRoot string, now time.Time) []fleetRow {
	settings := loadSystemSettings(dataRoot)
	threshold := float64(settings.AccountabilityMaxGapHours)

	cadence := map[string]float64{}
	if reg, err := withTaskSlots(dataRoot, func(*taskSlotRegistry) error { return nil }); err == nil {
		for _, t := range reg.Tasks {
			h := t.CadenceHours
			if t.Monthly {
				h = 24 * 31
			}
			if h > 0 && (cadence[t.ClientSlug] == 0 || h < cadence[t.ClientSlug]) {
				cadence[t.ClientSlug] = h
			}
		}
	}

	files, _ := filepath.Glob(filepath.Join(dataRoot, "fleet", "*.json"))
	var rows []fleetRow
	for _, p := range files {
		snap, err := readJSONFile(p)
		if err != nil {
			continue
		}
		slug := mStr(snap, "client_slug")
		if slug == "" {
			slug = strings.TrimSuffix(filepath.Base(p), ".json")
		}
		row := fleetRow{Client: slug, Name: mStr(snap, "client_name")}
		if row.Name == "" {
			row.Name = slug
		}

		posting := mMap(snap, "posting")
		acct := mMap(snap, "accountability")
		reminders := mInt(acct, "reminders_sent_this_episode", 0)
		escalated := mStr(acct, "operator_escalated_at") != ""
		gap, hasGap := posting["gap_hours"].(float64)
		lastPosted := mStr(posting, "last_posted_at")

		switch {
		case reminders >= 3 || escalated:
			row.Color, row.Rank = "err", 0
		case hasGap && threshold > 0 && gap >= threshold*2.0/3.0,
			mStr(posting, "status") == "warning", mStr(posting, "status") == "breach":
			row.Color, row.Rank = "warn", 1
		case lastPosted == "" && !hasGap:
			row.Color, row.Rank = "none", 2
		default:
			row.Color, row.Rank = "ok", 3
		}
		if hasGap {
			row.GapHours = gap
			row.LastPost = fmt.Sprintf("%.0fh ago", gap)
		} else if lastPosted != "" {
			row.LastPost = lastPosted
		} else {
			row.LastPost = "no posts yet"
		}
		if reminders > 0 {
			row.Reminders = fmt.Sprintf("reminded %d/3", reminders)
			if escalated {
				row.Reminders += " · operator emailed"
			}
		}

		totals := mMap(snap, "totals")
		row.Posts = fleetNum(totals, "posts_7d") + " / " + fleetNum(totals, "posts_30d")
		eng := mMap(snap, "engagement")
		row.Views7 = fleetNum(eng, "views_7d")
		row.Likes7 = fleetNum(eng, "likes_7d")
		leads := mMap(snap, "leads")
		row.Leads = fleetNum(leads, "hot") + " / " + fleetNum(leads, "warm")
		prod := mMap(snap, "production")
		row.IdeasQ = fleetNum(prod, "ideas_queued")
		row.DraftsPend = fleetNum(prod, "drafts_pending_approval")

		rep := mMap(snap, "report")
		if hp := mStr(rep, "html_path"); hp != "" {
			if rel, rerr := filepath.Rel(dataRoot, hp); rerr == nil && !strings.HasPrefix(rel, "..") {
				row.ReportHref = "/files/" + filepath.ToSlash(rel)
			}
		}

		if up := mStr(snap, "updated_at"); up != "" {
			if t, perr := time.Parse(time.RFC3339, up); perr == nil {
				age := now.Sub(t).Hours()
				row.Updated = fmt.Sprintf("%.0fh ago", age)
				cad := cadence[slug]
				if cad == 0 {
					cad = 33 // no registered task: assume daily-ish, flag past ~1.4 days
				}
				row.Stale = age > cad*1.5
			} else {
				row.Updated = up
			}
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Rank != rows[j].Rank {
			return rows[i].Rank < rows[j].Rank
		}
		return rows[i].GapHours > rows[j].GapHours
	})
	return rows
}

func (b *bridge) uiRenderFleet(w http.ResponseWriter) {
	rows := fleetRows(b.uiDataRoot, time.Now())
	tiles := map[string]int{"total": len(rows)}
	sumViews, sumPosts, sumDrafts := 0, 0, 0
	for _, r := range rows {
		switch r.Color {
		case "err":
			tiles["red"]++
		case "warn":
			tiles["yellow"]++
		case "ok":
			tiles["green"]++
		default:
			tiles["gray"]++
		}
		var v int
		if _, err := fmt.Sscanf(r.Views7, "%d", &v); err == nil {
			sumViews += v
		}
		var p int
		if _, err := fmt.Sscanf(r.Posts, "%d", &p); err == nil {
			sumPosts += p
		}
		var d int
		if _, err := fmt.Sscanf(r.DraftsPend, "%d", &d); err == nil {
			sumDrafts += d
		}
	}
	b.uiRender(w, "fleet", map[string]any{
		"Title": "Fleet", "Rows": rows, "Tiles": tiles,
		"SumViews": sumViews, "SumPosts": sumPosts, "SumDrafts": sumDrafts,
		"GapHours": loadSystemSettings(b.uiDataRoot).AccountabilityMaxGapHours,
	})
}
