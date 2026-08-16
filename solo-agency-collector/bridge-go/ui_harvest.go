package main

// ui_harvest.go — /ui/{client}/campaign/{slug}/harvest?stage=…: the drill-down
// behind every Progress tile of a Leads-From-Friends campaign. One row per
// profile at that stage, with whatever the system has collected for it —
// name, subtitle, seed, attempts, the enrich record's about/work/posts, the
// agent's verdict + reason, the CRM contact link. Read-only; every number on
// the campaign page must reconcile with a list here.

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type harvestRow struct {
	UID       string
	URL       string
	Name      string
	Subtitle  string
	Seed      string
	Status    string
	When      string
	Attempts  int
	Box       string
	Reason    string
	LeadID    string
	OK        string // "" | "ok" | "failed"
	Error     string
	Category  string
	About     []string
	Work      []string
	Posts     []map[string]string // {date, caption, url}
	Emails    []string
	Websites  []string
	Priority  string
	AvoidBox  string
}

var harvestStages = []struct{ Key, Label, Help string }{
	{"listed", "Friends listed", "every name read from the seeds' friend lists (client-wide registry, all statuses)"},
	{"known", "Skipped (already known)", "listed again but already in this client's registry — seed, queued elsewhere, or decided"},
	{"queued", "Queued to enrich", "waiting for the daemon; goal-keyword matches first"},
	{"in_flight", "Enriching now", "a collector job is open for these"},
	{"await", "Awaiting decision", "enriched; the agent judges these against the goal on the next run"},
	{"kept", "Kept → CRM", "matched the goal; a CRM contact exists"},
	{"rejected", "Rejected", "did not match the goal (reason recorded)"},
	{"failed", "Enrich failed", "unreadable after retries — remembered, never retried"},
	{"retried", "Retried", "re-queued after a stale/transient failure (attempts > 0)"},
}

func harvestStageLabel(key string) (string, string) {
	for _, s := range harvestStages {
		if s.Key == key {
			return s.Label, s.Help
		}
	}
	return key, ""
}

func strList(v any) []string {
	var out []string
	for _, x := range asSlice(v) {
		if s := strings.TrimSpace(sprint(x)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func agoOr(iso string, now time.Time) string {
	if iso == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	h := now.Sub(t).Hours()
	if h < 1 {
		return fmt.Sprintf("%.0f min ago", now.Sub(t).Minutes())
	}
	if h < 48 {
		return fmt.Sprintf("%.0fh ago", h)
	}
	return t.Format("2006-01-02")
}

// enrichRowFrom fills the collected-data half of a row from an enriched
// envelope (harvest/{campaign}/enriched/{uid_hash}.json).
func enrichRowFrom(row *harvestRow, env map[string]any) {
	if env == nil {
		return
	}
	if ok, has := env["ok"].(bool); has {
		if ok {
			row.OK = "ok"
		} else {
			row.OK = "failed"
			row.Error = mStr(env, "error")
		}
	}
	if row.Name == "" {
		row.Name = mStr(env, "name")
	}
	if row.Subtitle == "" {
		row.Subtitle = mStr(env, "subtitle")
	}
	row.Attempts = mInt(env, "attempts", row.Attempts)
	row.Box = strOr(mStr(env, "collector"), row.Box)
	rec := mMap(env, "record")
	if rec == nil {
		return
	}
	if row.Name == "" {
		row.Name = mStr(rec, "name")
	}
	row.Category = mStr(rec, "category")
	row.About = strList(rec["about_lines"])
	for _, w := range asSlice(rec["work"]) {
		if wm, ok := w.(map[string]any); ok {
			parts := []string{}
			for _, k := range []string{"title", "employer", "company", "location", "years", "text"} {
				if v := mStr(wm, k); v != "" {
					parts = append(parts, v)
				}
			}
			if len(parts) > 0 {
				row.Work = append(row.Work, strings.Join(parts, " · "))
			}
		} else if s := strings.TrimSpace(sprint(w)); s != "" {
			row.Work = append(row.Work, s)
		}
	}
	for _, p := range asSlice(rec["posts"]) {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		cap := strOr(mStr(pm, "caption"), strOr(mStr(pm, "text"), mStr(pm, "message")))
		if len(cap) > 240 {
			cap = cap[:240] + "…"
		}
		row.Posts = append(row.Posts, map[string]string{
			"date": strOr(mStr(pm, "date"), strOr(mStr(pm, "created_at"), mStr(pm, "time"))),
			"caption": cap,
			"url": strOr(mStr(pm, "url"), mStr(pm, "permalink")),
		})
	}
	row.Emails = strList(rec["emails"])
	row.Websites = strList(rec["websites"])
}

// harvestRows builds the drill-down list for one stage.
func (b *bridge) harvestRows(c uiClient, campaign, stage string, now time.Time) []harvestRow {
	outreachDir := filepath.Join(c.Path, "outreach")
	p, err := withProgress(outreachDir, campaign, func(*harvestProgress) error { return nil })
	if err != nil {
		return nil
	}
	reg, _ := withSeen(outreachDir, func(*seenRegistry) error { return nil })
	seen := map[string]seenProfile{}
	if reg != nil {
		seen = reg.Profiles
	}
	// Which seed uids belong to this campaign (for the "listed"/"known" scoping).
	seedOf := map[string]bool{}
	for _, s := range p.Seeds {
		seedOf[s.URL] = true
	}
	inCampaign := func(sp seenProfile) bool {
		return sp.Campaign == campaign || seedOf[sp.Seed]
	}
	var rows []harvestRow
	loadEnv := func(uid string) map[string]any {
		env, rerr := readJSONFile(harvestEnrichedPath(outreachDir, campaign, uid))
		if rerr != nil {
			return nil
		}
		return env
	}
	switch stage {
	case "queued", "retried":
		for _, q := range p.Queue {
			if stage == "retried" && q.Attempts == 0 {
				continue
			}
			pr := "later"
			if q.Priority == 0 {
				pr = "first (keyword match)"
			}
			rows = append(rows, harvestRow{UID: q.UID, URL: q.URL, Name: q.Name, Subtitle: q.Subtitle, Seed: q.Seed,
				Status: "queued", When: agoOr(q.QueuedAt, now), Attempts: q.Attempts, Priority: pr, AvoidBox: q.AvoidBox})
		}
	case "in_flight":
		for tag, f := range p.InFlight {
			st := "enriching"
			if strings.HasPrefix(tag, "leg:") {
				st = "reading friend list"
			}
			rows = append(rows, harvestRow{UID: tag, URL: f.URL, Name: f.Name, Subtitle: f.Subtitle, Seed: f.Seed,
				Status: st, When: agoOr(f.QueuedAt, now), Attempts: f.Attempts, Box: f.Box})
		}
	case "await":
		for _, uid := range p.AwaitDecision {
			sp := seen[uid]
			row := harvestRow{UID: uid, URL: sp.URL, Name: sp.Name, Seed: sp.Seed, Status: "awaiting decision", When: agoOr(sp.UpdatedAt, now)}
			enrichRowFrom(&row, loadEnv(uid))
			rows = append(rows, row)
		}
	case "kept", "rejected", "failed":
		want := map[string]string{"kept": "kept", "rejected": "rejected", "failed": "enrich_failed"}[stage]
		for uid, sp := range seen {
			if sp.Status != want || !inCampaign(sp) {
				continue
			}
			rows = append(rows, harvestRow{UID: uid, URL: sp.URL, Name: sp.Name, Seed: sp.Seed, Status: sp.Status,
				When: agoOr(sp.UpdatedAt, now), Reason: sp.Reason, LeadID: sp.LeadID})
		}
	case "known":
		// Everything in the registry that a leg of THIS campaign would skip: seeds,
		// and profiles decided/queued by any campaign of the client.
		for uid, sp := range seen {
			if sp.Status == "seed" || terminalSeen(sp.Status) || (sp.Status == "queued" && sp.Campaign != campaign) {
				rows = append(rows, harvestRow{UID: uid, URL: sp.URL, Name: sp.Name, Seed: sp.Seed, Status: sp.Status,
					When: agoOr(sp.UpdatedAt, now), Reason: sp.Reason, LeadID: sp.LeadID})
			}
		}
	default: // listed
		for uid, sp := range seen {
			if !inCampaign(sp) && sp.Status != "seed" {
				continue
			}
			row := harvestRow{UID: uid, URL: sp.URL, Name: sp.Name, Seed: sp.Seed, Status: sp.Status,
				When: agoOr(sp.UpdatedAt, now), Reason: sp.Reason, LeadID: sp.LeadID}
			if sp.Status == "enriched" {
				enrichRowFrom(&row, loadEnv(uid))
			}
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Status != rows[j].Status {
			return rows[i].Status < rows[j].Status
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

func (b *bridge) uiRenderHarvest(w http.ResponseWriter, slug, camp string, r *http.Request) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	store := newCrmStore(filepath.Join(c.Path, "outreach"))
	cfg := store.getCampaign(camp)
	if cfg == nil || mStr(cfg, "channel_strategy") != harvestChannel {
		http.NotFound(w, r)
		return
	}
	stage := r.URL.Query().Get("stage")
	if stage == "" {
		stage = "await"
	}
	label, help := harvestStageLabel(stage)
	rows := b.harvestRows(c, camp, stage, time.Now())
	b.uiRender(w, "harvest", map[string]any{
		"Title": camp + " · " + label, "Client": c, "Slug": camp, "Stage": stage,
		"StageLabel": label, "StageHelp": help, "Stages": harvestStages, "Rows": rows, "Count": len(rows),
		"Goal": mStr(mMap(cfg, "goal"), "description"),
	})
}
