package main

// schedule_slots.go — automation-task time-slot collision avoidance + the
// operator's global system settings.
//
// Clients run on mixed cadences (daily, 48h, 72h, 96h, weekly, monthly). When
// the agent creates a client's automation task it asks the human for a start
// time — but nothing checked how many OTHER tasks fire in that same slot on
// the days the new task will actually run. With mixed cadences that check
// cannot compare clock times: a 48h task and a 72h task at the same hour
// collide only every 6 days, a weekly task only on its weekday. The only
// correct check PROJECTS every task's future occurrences onto a timeline and
// counts concurrency at each occurrence of the NEW task. That math lives here
// in code (`tool schedule-slots suggest`), not in playbook prose — every
// brain doing its own LCM arithmetic is how rules drift.
//
// State:
//   - {pipeline}/automation/task_slots.json — machine-readable registry of
//     every automation task's run time + cadence (schedule.md and the
//     automation manifest stay the human-readable views).
//   - {pipeline}/system_settings.json — the operator's global system config,
//     editable in the web UI (/ui/settings) without chatting with an agent:
//     max concurrent tasks per slot (default 10), slot step (15m), horizon,
//     and the operator's contact email for future operator notifications.
//
// `suggest` walks candidate start times (requested, +step, +2·step, ...) and
// accepts the first one where NO future occurrence of the new task lands in a
// slot already holding max_concurrent_tasks running tasks.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// --- system settings -----------------------------------------------------------

type systemSettings struct {
	SchemaVersion          int    `json:"schema_version"`
	OperatorEmail          string `json:"operator_email"`
	MaxConcurrentTasks     int    `json:"max_concurrent_tasks"`
	SlotStepMinutes        int    `json:"slot_step_minutes"`
	SlotHorizonDays        int    `json:"slot_horizon_days"`
	DefaultTaskDurationMin int    `json:"default_task_duration_min"`
	// AccountabilityMaxGapHours: a client silent longer than this gets a
	// posting reminder (default 72h — a weekend gap is normal, Sat→Mon).
	AccountabilityMaxGapHours int `json:"accountability_max_gap_hours"`
	// --- distribution caps, per Facebook account, per day ------------------
	// The scarce resource in Messenger/comment outreach is ACCOUNT HEALTH, not
	// throughput: an email bounce costs one lead, but an account that looks like
	// a bulk sender gets restricted and takes its warmup history with it. These
	// are budgets spent down toward a cliff, so they start low and rise only on
	// evidence. Agency-wide ceilings; a campaign may ask for less, never more.
	DMPerAccountPerDay            int `json:"dm_per_account_per_day"`
	CommentGroupsPerAccountPerDay int `json:"comment_groups_per_account_per_day"`
	CommentsPerGroupPerDay        int `json:"comments_per_group_per_day"`
	// PostsPerAccountPerDay: posting INTO a group is the most exposed action of the
	// three — it is a standalone piece of content in front of the whole group, it is
	// what members report as spam, and many groups hold posts for admin approval. The
	// default is deliberately the lowest of the set.
	PostsPerAccountPerDay int `json:"posts_per_account_per_day"`
	// PublishGapMinutes: the floor between two collector-published actions for one
	// client (comment today, group post next). Approval is the command, so the
	// operator can approve six drafts in one sitting — this is what stops six
	// comments appearing within a minute, which is the shape Facebook reads as a
	// bot. It is a pacing floor, not a cap: the daily caps above are the ceiling.
	PublishGapMinutes int `json:"publish_gap_minutes"`
	// Friend-harvest pacing (Leads From Friends campaigns): profiles enriched
	// per day per campaign, per collector account per day, and the nightly
	// window in which the harvest daemon does not touch Facebook at all.
	HarvestDailyBudget        int    `json:"harvest_daily_budget"`
	HarvestPerCollectorBudget int    `json:"harvest_per_collector_budget"`
	HarvestQuietFrom          string `json:"harvest_quiet_from"`
	HarvestQuietTo            string `json:"harvest_quiet_to"`
	UpdatedAt                 string `json:"updated_at,omitempty"`
}

func defaultSystemSettings() systemSettings {
	return systemSettings{
		SchemaVersion:             1,
		MaxConcurrentTasks:        10,
		SlotStepMinutes:           15,
		SlotHorizonDays:           35,
		DefaultTaskDurationMin:    30,
		AccountabilityMaxGapHours: 72,
		// Operator-set 2026-08-15. 5 groups × 1 comment = 5 comments/account/day,
		// deliberately the same order of magnitude as the DM budget.
		DMPerAccountPerDay:            10,
		CommentGroupsPerAccountPerDay: 5,
		CommentsPerGroupPerDay:        1,
		PostsPerAccountPerDay:         2,
		PublishGapMinutes:             5,
		HarvestDailyBudget:            harvestDefaultDaily,
		HarvestPerCollectorBudget:     harvestDefaultPerBox,
		HarvestQuietFrom:              harvestDefaultQuietFrom,
		HarvestQuietTo:                harvestDefaultQuietTo,
	}
}

func systemSettingsPath(pipeline string) string {
	return filepath.Join(pipeline, "system_settings.json")
}

func loadSystemSettings(pipeline string) systemSettings {
	s := defaultSystemSettings()
	raw, err := os.ReadFile(systemSettingsPath(pipeline))
	if err != nil {
		return s
	}
	_ = json.Unmarshal(raw, &s)
	if s.MaxConcurrentTasks <= 0 {
		s.MaxConcurrentTasks = 10
	}
	if s.SlotStepMinutes <= 0 {
		s.SlotStepMinutes = 15
	}
	if s.PublishGapMinutes <= 0 {
		s.PublishGapMinutes = 5
	}
	if s.SlotHorizonDays <= 0 {
		s.SlotHorizonDays = 35
	}
	if s.DefaultTaskDurationMin <= 0 {
		s.DefaultTaskDurationMin = 30
	}
	if s.AccountabilityMaxGapHours <= 0 {
		s.AccountabilityMaxGapHours = 72
	}
	// A missing or zero distribution cap must fall back to the safe default, never
	// to "unlimited": an older settings file predating these fields would otherwise
	// read as 0 and a naive caller could treat that as no ceiling at all.
	if s.DMPerAccountPerDay <= 0 {
		s.DMPerAccountPerDay = 10
	}
	if s.CommentGroupsPerAccountPerDay <= 0 {
		s.CommentGroupsPerAccountPerDay = 5
	}
	if s.CommentsPerGroupPerDay <= 0 {
		s.CommentsPerGroupPerDay = 1
	}
	if s.PostsPerAccountPerDay <= 0 {
		s.PostsPerAccountPerDay = 2
	}
	if s.HarvestDailyBudget <= 0 {
		s.HarvestDailyBudget = harvestDefaultDaily
	}
	if s.HarvestPerCollectorBudget <= 0 {
		s.HarvestPerCollectorBudget = harvestDefaultPerBox
	}
	if !runTimeRe.MatchString(s.HarvestQuietFrom) {
		s.HarvestQuietFrom = harvestDefaultQuietFrom
	}
	if !runTimeRe.MatchString(s.HarvestQuietTo) {
		s.HarvestQuietTo = harvestDefaultQuietTo
	}
	return s
}

func saveSystemSettings(pipeline string, mutate func(*systemSettings) error) (systemSettings, error) {
	s := defaultSystemSettings()
	err := withLockedJSON(systemSettingsPath(pipeline),
		func(raw []byte) error {
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &s); err != nil {
					return fmt.Errorf("system_settings.json is corrupt: %w", err)
				}
			}
			if s.SchemaVersion == 0 {
				s.SchemaVersion = 1
			}
			return nil
		},
		func() ([]byte, error) { return json.MarshalIndent(&s, "", "  ") },
		func() error {
			if err := mutate(&s); err != nil {
				return err
			}
			s.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			return nil
		})
	return s, err
}

// --- task slot registry --------------------------------------------------------

type taskSlot struct {
	TaskName     string  `json:"task_name"`
	ClientSlug   string  `json:"client_slug"`
	CadenceHours float64 `json:"cadence_hours,omitempty"` // 24, 48, 72, 96, 168...
	Monthly      bool    `json:"monthly,omitempty"`       // calendar-month cadence (same day-of-month)
	RunTime      string  `json:"run_time"`                // "HH:MM" local
	AnchorDate   string  `json:"anchor_date"`             // "YYYY-MM-DD" local first-run date
	DurationMin  int     `json:"duration_min"`
	Status       string  `json:"status"` // active | paused
	RegisteredAt string  `json:"registered_at"`
	UpdatedAt    string  `json:"updated_at,omitempty"`
}

type taskSlotRegistry struct {
	SchemaVersion int        `json:"schema_version"`
	Tasks         []taskSlot `json:"tasks"`
}

func taskSlotsPath(pipeline string) string {
	return filepath.Join(pipeline, "automation", "task_slots.json")
}

func withTaskSlots(pipeline string, fn func(*taskSlotRegistry) error) (*taskSlotRegistry, error) {
	reg := &taskSlotRegistry{}
	err := withLockedJSON(taskSlotsPath(pipeline),
		func(raw []byte) error {
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, reg); err != nil {
					return fmt.Errorf("task_slots.json is corrupt: %w", err)
				}
			}
			if reg.SchemaVersion == 0 {
				reg.SchemaVersion = 1
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

var runTimeRe = regexp.MustCompile(`^([01]?\d|2[0-3]):([0-5]\d)$`)
var anchorDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// firstStart resolves a slot's first occurrence in local time.
func slotFirstStart(anchorDate, runTime string, loc *time.Location) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02 15:04", anchorDate+" "+runTime, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("bad anchor/time %q %q: %w", anchorDate, runTime, err)
	}
	return t, nil
}

// occurrences lists a task's start times inside [from, to).
func slotOccurrences(s taskSlot, from, to time.Time, loc *time.Location) []time.Time {
	first, err := slotFirstStart(s.AnchorDate, s.RunTime, loc)
	if err != nil {
		return nil
	}
	var out []time.Time
	if s.Monthly {
		t := first
		for t.Before(to) {
			if !t.Before(from) {
				out = append(out, t)
			}
			t = t.AddDate(0, 1, 0)
		}
		return out
	}
	cad := s.CadenceHours
	if cad <= 0 {
		cad = 24
	}
	step := time.Duration(cad * float64(time.Hour))
	t := first
	// Jump close to `from` instead of iterating from a months-old anchor.
	if t.Before(from) {
		k := int64(from.Sub(t) / step)
		t = t.Add(time.Duration(k) * step)
		for t.Before(from) {
			t = t.Add(step)
		}
	}
	for t.Before(to) {
		out = append(out, t)
		t = t.Add(step)
	}
	return out
}

type slotConflict struct {
	At    string   `json:"at"`
	Count int      `json:"count"`
	Tasks []string `json:"tasks"`
}

// worstConflict finds, across every occurrence of the candidate task, the slot
// with the highest number of EXISTING active tasks whose run interval overlaps
// the candidate's [start, start+duration).
func worstConflict(cand taskSlot, existing []taskSlot, settings systemSettings, from, to time.Time, loc *time.Location) slotConflict {
	worst := slotConflict{}
	candDur := time.Duration(cand.DurationMin) * time.Minute
	for _, occ := range slotOccurrences(cand, from, to, loc) {
		occEnd := occ.Add(candDur)
		count := 0
		var names []string
		for _, ex := range existing {
			if ex.Status == "paused" || ex.TaskName == cand.TaskName {
				continue
			}
			exDur := time.Duration(ex.DurationMin) * time.Minute
			if exDur <= 0 {
				exDur = time.Duration(settings.DefaultTaskDurationMin) * time.Minute
			}
			for _, exOcc := range slotOccurrences(ex, occ.Add(-exDur), occEnd, loc) {
				if exOcc.Before(occEnd) && exOcc.Add(exDur).After(occ) {
					count++
					names = append(names, ex.TaskName)
					break
				}
			}
		}
		if count > worst.Count {
			worst = slotConflict{At: occ.Format("2006-01-02 15:04"), Count: count, Tasks: names}
		}
	}
	return worst
}

// --- CLI: tool schedule-slots --------------------------------------------------

func scheduleSlotsUsage() int {
	fmt.Fprintln(os.Stderr, `usage: tool schedule-slots --pipeline DIR <subcommand>
  suggest  --task NAME --client S --time "HH:MM" [--cadence-hours 48 | --cadence monthly] [--anchor-date YYYY-MM-DD] [--duration-min 30]
           -> first start time (requested, +step, +2*step, ...) where no future occurrence
              exceeds max_concurrent_tasks (system_settings.json, default 10)
  register --task NAME --client S --time "HH:MM" [--cadence-hours H | --cadence monthly] [--anchor-date D] [--duration-min M] [--status active|paused]
  remove   --task NAME
  list`)
	return 2
}

func runScheduleSlotsCLI(args []string) int {
	valueFlags := map[string]bool{"--pipeline": true, "--task": true, "--client": true,
		"--time": true, "--cadence-hours": true, "--cadence": true, "--anchor-date": true,
		"--duration-min": true, "--status": true}
	a, err := parseCLIArgs(args, valueFlags, map[string]bool{})
	if err != nil {
		return crmUsageErr(err.Error())
	}
	if len(a.pos) == 0 {
		return scheduleSlotsUsage()
	}
	pipeline := a.get("--pipeline")
	if pipeline == "" {
		return crmUsageErr("--pipeline DIR is required")
	}
	if abs, aerr := filepath.Abs(pipeline); aerr == nil {
		pipeline = abs
	}
	loc := time.Local
	now := time.Now().In(loc)
	nowStr := now.UTC().Format(time.RFC3339)

	buildSlot := func() (taskSlot, int) {
		s := taskSlot{
			TaskName:   strings.TrimSpace(a.get("--task")),
			ClientSlug: strings.TrimSpace(a.get("--client")),
			RunTime:    strings.TrimSpace(a.get("--time")),
			AnchorDate: strings.TrimSpace(a.get("--anchor-date")),
			Status:     "active",
		}
		if s.AnchorDate == "" {
			s.AnchorDate = now.Format("2006-01-02")
		}
		if !runTimeRe.MatchString(s.RunTime) {
			return s, crmUsageErr(`--time must be "HH:MM" (24h)`)
		}
		if !anchorDateRe.MatchString(s.AnchorDate) {
			return s, crmUsageErr(`--anchor-date must be "YYYY-MM-DD"`)
		}
		cadWord := strings.ToLower(strings.TrimSpace(a.get("--cadence")))
		switch cadWord {
		case "monthly":
			s.Monthly = true
		case "", "hours":
			s.CadenceHours = float64(a.getInt("--cadence-hours", 24))
			if s.CadenceHours <= 0 {
				return s, crmUsageErr("--cadence-hours must be a positive number of hours")
			}
		case "daily":
			s.CadenceHours = 24
		case "weekly":
			s.CadenceHours = 168
		default:
			return s, crmUsageErr("--cadence must be daily, weekly, monthly, or omit it and pass --cadence-hours")
		}
		s.DurationMin = a.getInt("--duration-min", 0)
		if s.DurationMin <= 0 {
			s.DurationMin = loadSystemSettings(pipeline).DefaultTaskDurationMin
		}
		return s, -1
	}

	switch a.pos[0] {
	case "suggest":
		cand, bad := buildSlot()
		if bad >= 0 {
			return bad
		}
		if cand.TaskName == "" {
			return crmUsageErr("suggest needs --task (the collision check must be able to exclude the task's own old slot)")
		}
		settings := loadSystemSettings(pipeline)
		var existing []taskSlot
		if reg, rerr := withTaskSlots(pipeline, func(*taskSlotRegistry) error { return nil }); rerr == nil {
			existing = reg.Tasks
		}
		from := now
		to := now.AddDate(0, 0, settings.SlotHorizonDays)
		requested := cand.RunTime
		base, ferr := slotFirstStart(cand.AnchorDate, cand.RunTime, loc)
		if ferr != nil {
			return crmFail(ferr)
		}
		var lastConflict slotConflict
		// 96 steps of 15m = a full day of shifting; if nothing frees up by then
		// the limit is set below the real client count and needs raising.
		for step := 0; step <= 96; step++ {
			shifted := base.Add(time.Duration(step*settings.SlotStepMinutes) * time.Minute)
			cand.AnchorDate = shifted.Format("2006-01-02")
			cand.RunTime = shifted.Format("15:04")
			worst := worstConflict(cand, existing, settings, from, to, loc)
			if worst.Count < settings.MaxConcurrentTasks {
				return crmOut(map[string]any{
					"ok": true, "requested": requested, "suggested_time": cand.RunTime,
					"suggested_anchor_date": cand.AnchorDate,
					"shifted_minutes":       step * settings.SlotStepMinutes,
					"max_concurrent_tasks":  settings.MaxConcurrentTasks,
					"horizon_days":          settings.SlotHorizonDays,
					"busiest_slot":          worst,
				}, 0)
			}
			lastConflict = worst
		}
		return crmFail(fmt.Errorf("no free slot within 24h of %s: every slot holds >= %d tasks (busiest: %s with %d) — raise max_concurrent_tasks in /ui/settings or spread cadences",
			requested, settings.MaxConcurrentTasks, lastConflict.At, lastConflict.Count))

	case "register":
		s, bad := buildSlot()
		if bad >= 0 {
			return bad
		}
		if s.TaskName == "" || s.ClientSlug == "" {
			return crmUsageErr("register needs --task and --client")
		}
		if st := strings.ToLower(strings.TrimSpace(a.get("--status"))); st != "" {
			if st != "active" && st != "paused" {
				return crmUsageErr("--status must be active or paused")
			}
			s.Status = st
		}
		s.RegisteredAt = nowStr
		s.UpdatedAt = nowStr
		_, err := withTaskSlots(pipeline, func(reg *taskSlotRegistry) error {
			for i := range reg.Tasks {
				if reg.Tasks[i].TaskName == s.TaskName {
					s.RegisteredAt = reg.Tasks[i].RegisteredAt
					reg.Tasks[i] = s
					return nil
				}
			}
			reg.Tasks = append(reg.Tasks, s)
			return nil
		})
		if err != nil {
			return crmFail(err)
		}
		return crmOut(map[string]any{"ok": true, "task": s}, 0)

	case "remove":
		name := strings.TrimSpace(a.get("--task"))
		if name == "" {
			return crmUsageErr("remove needs --task")
		}
		removed := false
		_, err := withTaskSlots(pipeline, func(reg *taskSlotRegistry) error {
			kept := reg.Tasks[:0]
			for _, t := range reg.Tasks {
				if t.TaskName == name {
					removed = true
					continue
				}
				kept = append(kept, t)
			}
			reg.Tasks = kept
			return nil
		})
		if err != nil {
			return crmFail(err)
		}
		return crmOut(map[string]any{"ok": true, "removed": removed}, 0)

	case "list":
		reg, err := withTaskSlots(pipeline, func(*taskSlotRegistry) error { return nil })
		if err != nil {
			return crmFail(err)
		}
		sort.Slice(reg.Tasks, func(i, j int) bool { return reg.Tasks[i].TaskName < reg.Tasks[j].TaskName })
		return crmOut(map[string]any{"ok": true, "count": len(reg.Tasks), "tasks": reg.Tasks,
			"settings": loadSystemSettings(pipeline)}, 0)
	}
	return scheduleSlotsUsage()
}

// --- CLI: tool system-settings -------------------------------------------------

func runSystemSettingsCLI(args []string) int {
	valueFlags := map[string]bool{"--pipeline": true, "--email": true, "--max-concurrent": true,
		"--step-minutes": true, "--horizon-days": true, "--default-duration-min": true,
		"--accountability-gap-hours": true}
	a, err := parseCLIArgs(args, valueFlags, map[string]bool{})
	if err != nil {
		return crmUsageErr(err.Error())
	}
	usage := func() int {
		fmt.Fprintln(os.Stderr, `usage: tool system-settings --pipeline DIR get
       tool system-settings --pipeline DIR set [--email X] [--max-concurrent N] [--step-minutes N] [--horizon-days N] [--default-duration-min N]`)
		return 2
	}
	if len(a.pos) == 0 {
		return usage()
	}
	pipeline := a.get("--pipeline")
	if pipeline == "" {
		return crmUsageErr("--pipeline DIR is required")
	}
	if abs, aerr := filepath.Abs(pipeline); aerr == nil {
		pipeline = abs
	}
	switch a.pos[0] {
	case "get":
		return crmOut(loadSystemSettings(pipeline), 0)
	case "set":
		s, err := saveSystemSettings(pipeline, func(s *systemSettings) error {
			if v := strings.TrimSpace(a.get("--email")); v != "" {
				if !strings.Contains(v, "@") {
					return fmt.Errorf("not an email address: %q", v)
				}
				s.OperatorEmail = v
			}
			if v := a.getInt("--max-concurrent", 0); v > 0 {
				s.MaxConcurrentTasks = v
			}
			if v := a.getInt("--step-minutes", 0); v > 0 {
				s.SlotStepMinutes = v
			}
			if v := a.getInt("--horizon-days", 0); v > 0 {
				s.SlotHorizonDays = v
			}
			if v := a.getInt("--default-duration-min", 0); v > 0 {
				s.DefaultTaskDurationMin = v
			}
			if v := a.getInt("--accountability-gap-hours", 0); v > 0 {
				s.AccountabilityMaxGapHours = v
			}
			return nil
		})
		if err != nil {
			return crmFail(err)
		}
		return crmOut(s, 0)
	}
	return usage()
}
