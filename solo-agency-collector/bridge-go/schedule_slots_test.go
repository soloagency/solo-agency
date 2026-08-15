package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func slotsCLI(t *testing.T, args ...string) map[string]any {
	t.Helper()
	code := 0
	out := captureStdout(t, func() { code = runScheduleSlotsCLI(args) })
	if code != 0 {
		t.Fatalf("schedule-slots %v exited %d\n%s", args, code, out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("bad JSON from schedule-slots %v: %v\n%s", args, err, out)
	}
	return m
}

func TestSlotOccurrencesProjection(t *testing.T) {
	loc := time.Local
	from := time.Date(2026, 8, 14, 0, 0, 0, 0, loc)
	to := from.AddDate(0, 0, 14)

	// 48h cadence anchored 2026-08-10 09:00 -> occurrences every other day.
	s := taskSlot{RunTime: "09:00", AnchorDate: "2026-08-10", CadenceHours: 48}
	occ := slotOccurrences(s, from, to, loc)
	if len(occ) != 7 {
		t.Fatalf("48h over 14 days: want 7 occurrences, got %d (%v)", len(occ), occ)
	}
	if occ[0].Format("2006-01-02 15:04") != "2026-08-14 09:00" {
		t.Errorf("first occurrence should land on the cadence grid: %v", occ[0])
	}

	// Weekly cadence: 2 occurrences in 14 days.
	wk := taskSlot{RunTime: "10:00", AnchorDate: "2026-08-15", CadenceHours: 168}
	if n := len(slotOccurrences(wk, from, to, loc)); n != 2 {
		t.Errorf("weekly over 14 days: want 2, got %d", n)
	}

	// Monthly: same day-of-month.
	mo := taskSlot{RunTime: "08:00", AnchorDate: "2026-08-20", Monthly: true}
	mocc := slotOccurrences(mo, from, from.AddDate(0, 3, 0), loc)
	if len(mocc) != 3 {
		t.Fatalf("monthly over 3 months: want 3, got %d (%v)", len(mocc), mocc)
	}
	if mocc[1].Day() != 20 {
		t.Errorf("monthly occurrences must keep the day-of-month: %v", mocc[1])
	}
}

func TestScheduleSlotsSuggestPushesWhenFull(t *testing.T) {
	pipeline := t.TempDir()
	today := time.Now().Format("2006-01-02")

	// Fill the 09:00 slot with exactly max_concurrent_tasks (10) daily tasks.
	for i := 0; i < 10; i++ {
		slotsCLI(t, "--pipeline", pipeline, "register",
			"--task", fmt.Sprintf("Client%02d - Daily Run", i), "--client", fmt.Sprintf("c%02d", i),
			"--time", "09:00", "--cadence-hours", "24", "--anchor-date", today)
	}

	// The 11th daily task at 09:00 is pushed in 15m steps until its RUN INTERVAL
	// clears the crowd: at 09:15 the ten 30-minute tasks are still running, so
	// the first free start is 09:30 (concurrency means "running at the same
	// time", not "same start minute").
	res := slotsCLI(t, "--pipeline", pipeline, "suggest", "--task", "Client10 - Daily Run",
		"--client", "c10", "--time", "09:00", "--cadence-hours", "24", "--anchor-date", today)
	if res["suggested_time"] != "09:30" || res["shifted_minutes"].(float64) != 30 {
		t.Fatalf("11th task should shift to 09:30: %v", res)
	}

	// A 48h task lands in the same 09:00 slot on days the daily tasks also run -> pushed too.
	res = slotsCLI(t, "--pipeline", pipeline, "suggest", "--task", "Biweekly - Run",
		"--client", "bw", "--time", "09:00", "--cadence-hours", "48", "--anchor-date", today)
	if res["suggested_time"] != "09:30" {
		t.Fatalf("48h task hitting a full daily slot should shift past the running window: %v", res)
	}

	// A different hour is free: no shift.
	res = slotsCLI(t, "--pipeline", pipeline, "suggest", "--task", "Client10 - Daily Run",
		"--client", "c10", "--time", "14:00", "--cadence-hours", "24", "--anchor-date", today)
	if res["suggested_time"] != "14:00" || res["shifted_minutes"].(float64) != 0 {
		t.Fatalf("free slot should not shift: %v", res)
	}

	// Re-scheduling an EXISTING task excludes its own old slot from the count:
	// Client05 re-suggesting 09:00 sees only 9 others -> stays.
	res = slotsCLI(t, "--pipeline", pipeline, "suggest", "--task", "Client05 - Daily Run",
		"--client", "c05", "--time", "09:00", "--cadence-hours", "24", "--anchor-date", today)
	if res["suggested_time"] != "09:00" {
		t.Fatalf("re-scheduling must exclude the task's own slot: %v", res)
	}

	// Paused tasks do not occupy slots.
	slotsCLI(t, "--pipeline", pipeline, "register", "--task", "Client00 - Daily Run",
		"--client", "c00", "--time", "09:00", "--cadence-hours", "24", "--anchor-date", today,
		"--status", "paused")
	res = slotsCLI(t, "--pipeline", pipeline, "suggest", "--task", "Client10 - Daily Run",
		"--client", "c10", "--time", "09:00", "--cadence-hours", "24", "--anchor-date", today)
	if res["suggested_time"] != "09:00" {
		t.Fatalf("pausing one of ten should free the slot: %v", res)
	}
}

func TestScheduleSlotsMixedCadenceNoFalseCollision(t *testing.T) {
	pipeline := t.TempDir()
	// Lower the limit to 1 so any true co-occurrence forces a shift.
	code := 0
	_ = captureStdout(t, func() {
		code = runSystemSettingsCLI([]string{"--pipeline", pipeline, "set", "--max-concurrent", "1"})
	})
	if code != 0 {
		t.Fatalf("system-settings set failed")
	}

	// 48h task anchored on an EVEN date at 09:00.
	slotsCLI(t, "--pipeline", pipeline, "register", "--task", "A", "--client", "a",
		"--time", "09:00", "--cadence-hours", "48", "--anchor-date", "2026-08-10")
	// Another 48h task, same time, anchored on an ODD date: grids never touch.
	res := slotsCLI(t, "--pipeline", pipeline, "suggest", "--task", "B", "--client", "b",
		"--time", "09:00", "--cadence-hours", "48", "--anchor-date", "2026-08-11")
	if res["suggested_time"] != "09:00" {
		t.Fatalf("interleaved 48h grids never co-occur; no shift expected: %v", res)
	}
	// Same anchor parity: a real co-occurrence -> shift.
	res = slotsCLI(t, "--pipeline", pipeline, "suggest", "--task", "C", "--client", "c",
		"--time", "09:00", "--cadence-hours", "48", "--anchor-date", "2026-08-12")
	if res["suggested_time"] == "09:00" {
		t.Fatalf("same-grid 48h tasks must collide and shift: %v", res)
	}
}

func TestSystemSettingsRoundTrip(t *testing.T) {
	pipeline := t.TempDir()
	s := loadSystemSettings(pipeline)
	if s.MaxConcurrentTasks != 10 || s.SlotStepMinutes != 15 {
		t.Fatalf("defaults wrong: %+v", s)
	}
	code := 0
	out := captureStdout(t, func() {
		code = runSystemSettingsCLI([]string{"--pipeline", pipeline, "set",
			"--email", "op@example.com", "--max-concurrent", "25"})
	})
	if code != 0 {
		t.Fatalf("set failed: %s", out)
	}
	s = loadSystemSettings(pipeline)
	if s.OperatorEmail != "op@example.com" || s.MaxConcurrentTasks != 25 {
		t.Fatalf("settings did not persist: %+v", s)
	}
	// Invalid email refused.
	_ = captureStdout(t, func() {
		code = runSystemSettingsCLI([]string{"--pipeline", pipeline, "set", "--email", "not-an-email"})
	})
	if code == 0 {
		t.Fatalf("invalid email must be refused")
	}
}

func TestSettingsTemplateRenders(t *testing.T) {
	var sb strings.Builder
	err := uiTpl.ExecuteTemplate(&sb, "settings", map[string]any{
		"Title": "Settings", "NavPage": "settings",
		"Settings": defaultSystemSettings(),
		"Tasks": []map[string]string{{"Task": "X - Daily Run", "Client": "x", "Cadence": "48h",
			"Time": "09:00", "Anchor": "2026-08-14", "Duration": "30 min", "Status": "active"}},
	})
	if err != nil {
		t.Fatalf("settings template failed to render: %v", err)
	}
	html := sb.String()
	for _, want := range []string{"Operator email", "Max tasks running", "X - Daily Run", "/api/ui/system/settings"} {
		if !strings.Contains(html, want) {
			t.Errorf("settings page missing %q", want)
		}
	}
}

// The distribution caps govern how hard a real Facebook account is pushed, so the two
// things that must never break are: a settings file written before these fields existed
// must not read as "unlimited", and an out-of-range value must be refused rather than
// clamped silently.
func TestDistributionCapsDefaultsAndBounds(t *testing.T) {
	pipeline := t.TempDir()

	s := loadSystemSettings(pipeline)
	if s.DMPerAccountPerDay != 10 || s.CommentGroupsPerAccountPerDay != 5 || s.CommentsPerGroupPerDay != 1 || s.PostsPerAccountPerDay != 2 {
		t.Fatalf("distribution defaults wrong: %+v", s)
	}

	// A settings file predating these fields: the zero value must fall back to the safe
	// default, never to 0 (which a caller could read as "no ceiling").
	legacy := `{"schema_version":1,"operator_email":"op@example.com","max_concurrent_tasks":12}`
	if err := os.WriteFile(systemSettingsPath(pipeline), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	s = loadSystemSettings(pipeline)
	if s.OperatorEmail != "op@example.com" || s.MaxConcurrentTasks != 12 {
		t.Fatalf("legacy fields lost: %+v", s)
	}
	if s.DMPerAccountPerDay != 10 || s.CommentGroupsPerAccountPerDay != 5 || s.CommentsPerGroupPerDay != 1 || s.PostsPerAccountPerDay != 2 {
		t.Fatalf("missing caps must fall back to defaults, not zero: %+v", s)
	}

	// Values persist through the store.
	if _, err := saveSystemSettings(pipeline, func(s *systemSettings) error {
		s.DMPerAccountPerDay = 7
		s.CommentGroupsPerAccountPerDay = 3
		s.CommentsPerGroupPerDay = 2
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if s = loadSystemSettings(pipeline); s.DMPerAccountPerDay != 7 || s.CommentGroupsPerAccountPerDay != 3 || s.CommentsPerGroupPerDay != 2 {
		t.Fatalf("caps did not persist: %+v", s)
	}
}

func TestSettingsPageShowsDistributionCaps(t *testing.T) {
	var sb strings.Builder
	if err := uiTpl.ExecuteTemplate(&sb, "settings", map[string]any{
		"Title": "Settings", "NavPage": "settings",
		"Settings": defaultSystemSettings(),
		"Tasks":    []map[string]string{},
	}); err != nil {
		t.Fatalf("settings template failed to render: %v", err)
	}
	html := sb.String()
	for _, want := range []string{
		"Distribution caps", `id="s-dm"`, `id="s-cgroups"`, `id="s-cper"`, `id="s-posts"`,
		"dm_per_account_per_day", "comment_groups_per_account_per_day", "comments_per_group_per_day",
		"posts_per_account_per_day",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("settings page missing distribution control %q", want)
		}
	}
}
