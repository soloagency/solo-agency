package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFleetRowsRankingAndColors(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	iso := func(hoursAgo float64) string {
		return now.Add(-time.Duration(hoursAgo * float64(time.Hour))).Format(time.RFC3339)
	}
	writeJSONT(t, systemSettingsPath(root), map[string]any{"accountability_max_gap_hours": 72})
	writeJSONT(t, taskSlotsPath(root), map[string]any{"schema_version": 1, "tasks": []any{
		map[string]any{"task_name": "Green - Run", "client_slug": "green", "cadence_hours": 48,
			"run_time": "09:00", "anchor_date": "2026-08-01", "duration_min": 30, "status": "active"},
	}})

	writeJSONT(t, filepath.Join(root, "fleet", "red.json"), map[string]any{
		"client_slug": "red", "client_name": "Red Client", "updated_at": iso(1),
		"posting":        map[string]any{"last_posted_at": iso(200), "gap_hours": 200.0, "status": "breach"},
		"accountability": map[string]any{"reminders_sent_this_episode": 3, "operator_escalated_at": iso(2)},
	})
	writeJSONT(t, filepath.Join(root, "fleet", "yellow.json"), map[string]any{
		"client_slug": "yellow", "updated_at": iso(1),
		"posting": map[string]any{"last_posted_at": iso(60), "gap_hours": 60.0, "status": "ok"},
		"totals":  map[string]any{"posts_7d": 3.0, "posts_30d": 12.0},
	})
	writeJSONT(t, filepath.Join(root, "fleet", "gray.json"), map[string]any{
		"client_slug": "gray", "updated_at": iso(1),
		"posting": map[string]any{},
	})
	// Green client's snapshot is 100h old with a 48h cadence -> stale.
	writeJSONT(t, filepath.Join(root, "fleet", "green.json"), map[string]any{
		"client_slug": "green", "updated_at": iso(100),
		"posting":    map[string]any{"last_posted_at": iso(10), "gap_hours": 10.0, "status": "ok"},
		"engagement": map[string]any{"views_7d": 1234.0},
	})

	rows := fleetRows(root, now)
	if len(rows) != 4 {
		t.Fatalf("want 4 rows, got %d", len(rows))
	}
	order := []string{rows[0].Client, rows[1].Client, rows[2].Client, rows[3].Client}
	if order[0] != "red" || order[1] != "yellow" || order[2] != "gray" || order[3] != "green" {
		t.Fatalf("worst-first order wrong: %v", order)
	}
	if rows[0].Color != "err" || !strings.Contains(rows[0].Reminders, "3/3") {
		t.Fatalf("red row wrong: %+v", rows[0])
	}
	// 60h gap with 72h threshold: >= 2/3 (48h) -> yellow even though status says ok.
	if rows[1].Color != "warn" {
		t.Fatalf("yellow row wrong: %+v", rows[1])
	}
	if rows[2].Color != "none" || rows[2].LastPost != "no posts yet" {
		t.Fatalf("gray row wrong: %+v", rows[2])
	}
	if rows[3].Color != "ok" || !rows[3].Stale {
		t.Fatalf("green row should be ok + stale (100h old snapshot, 48h cadence): %+v", rows[3])
	}
	if rows[3].Views7 != "1234" {
		t.Fatalf("views not carried: %+v", rows[3])
	}
}

func TestFleetTemplateRenders(t *testing.T) {
	var sb strings.Builder
	err := uiTpl.ExecuteTemplate(&sb, "fleet", map[string]any{
		"Title": "Fleet", "NavPage": "fleet",
		"Tiles":    map[string]int{"total": 2, "green": 1, "yellow": 0, "red": 1, "gray": 0},
		"SumViews": 1234, "SumPosts": 5, "SumDrafts": 2, "GapHours": 72,
		"Rows": []fleetRow{
			{Client: "red", Name: "Red Client", Color: "err", LastPost: "200h ago",
				Reminders: "reminded 3/3 · operator emailed", Posts: "0 / 2", Views7: "10",
				Likes7: "1", Leads: "0 / 1", IdeasQ: "3", DraftsPend: "2", Updated: "1h ago"},
			{Client: "green", Name: "Green", Color: "ok", LastPost: "10h ago", Posts: "3 / 12",
				Views7: "1234", Likes7: "99", Leads: "1 / 2", IdeasQ: "6", DraftsPend: "0",
				ReportHref: "/files/outputs/x.html", Updated: "100h ago", Stale: true},
		},
	})
	if err != nil {
		t.Fatalf("fleet template failed: %v", err)
	}
	html := sb.String()
	for _, want := range []string{"Red Client", "reminded 3/3", "stale", "/files/outputs/x.html", "red alerts"} {
		if !strings.Contains(html, want) {
			t.Errorf("fleet page missing %q", want)
		}
	}
}
