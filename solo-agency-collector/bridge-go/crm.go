package main

// crm.go — Go port of outreach/tools/crm_store.py (core: scaffold, pipelines,
// segments, campaigns, enrich queue, validation, enrichment, contacts/identity/
// merge, activities/tasks/deals, suppression, deterministic rules engine).
// Drafts/approvals/reports live in crm_reports.go; the CLI in crm_cli.go.
//
// Porting rule: same files, same semantics, same error strings (tests and
// playbooks match on them). Records are map[string]any so unknown fields
// survive read-modify-write; key order in output files may differ from Python
// (readers parse, never byte-compare).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func defaultPipelines() map[string]any {
	var m map[string]any
	_ = json.Unmarshal([]byte(defaultPipelinesJSON), &m)
	return m
}

// Kept as JSON so the Go and Python defaults cannot drift in structure.
const defaultPipelinesJSON = `{
  "pipelines": [{"id": "default_sales", "stages": [
    {"id": "new_reply", "probability": 0.10, "sla_days": 1},
    {"id": "engaged", "probability": 0.25, "sla_days": 7},
    {"id": "meeting_booked", "probability": 0.50, "sla_days": 7},
    {"id": "proposal_sent", "probability": 0.70, "sla_days": 10},
    {"id": "won"}, {"id": "lost"}
  ]}],
  "rules": [
    {"id": "r1", "on": "reply_positive", "do": ["create_deal_if_none(stage=new_reply)", "create_task(title=Reply within 4h,due=+4h)", "freeze_sequence"]},
    {"id": "r2", "on": "reply_question", "do": ["create_deal_if_none(stage=engaged)", "freeze_sequence", "draft_reply_for_approval"]},
    {"id": "r3", "on": "reply_negative|remove_intent", "do": ["suppress(contact)", "freeze_sequence", "close_open_tasks"]},
    {"id": "r4", "on": "stage_age_exceeds_sla", "do": ["create_task(nudge)", "flag_in_report"]},
    {"id": "r5", "on": "deal_won", "do": ["set_lifecycle(customer)", "enroll_segment(customers)", "create_task(onboarding)"]},
    {"id": "r6", "on": "hard_bounce|unsubscribe", "do": ["suppress(contact)", "freeze_sequence", "close_open_tasks"]}
  ]
}`

type crmStore struct {
	clientDir string
	crmRoot   string
	a         *jsonStore
}

func newCrmStore(clientDir string) *crmStore {
	abs, err := filepath.Abs(clientDir)
	if err != nil {
		abs = clientDir
	}
	return &crmStore{clientDir: abs, crmRoot: filepath.Join(abs, "crm"), a: newJSONStore(abs)}
}

// --- scaffold -----------------------------------------------------------------

func (c *crmStore) initTree() error {
	for _, sub := range []string{"contacts", "accounts", "deals", "activities", "tasks", "segments", "reports"} {
		if err := os.MkdirAll(filepath.Join(c.crmRoot, sub), 0o755); err != nil {
			return err
		}
	}
	for _, sub := range []string{"sendboxes", "lists", "campaigns", "approvals", "analytics",
		"inbox_sync", "integrations/providers", "outputs"} {
		if err := os.MkdirAll(filepath.Join(c.clientDir, filepath.FromSlash(sub)), 0o755); err != nil {
			return err
		}
	}
	_, err := c.ensureDefaultPipelines()
	return err
}

// --- pipelines ----------------------------------------------------------------

func (c *crmStore) pipelinesPath() string { return filepath.Join(c.crmRoot, "pipelines.json") }

func (c *crmStore) getPipelines() map[string]any {
	if m, err := readJSONFile(c.pipelinesPath()); err == nil {
		return m
	}
	return map[string]any{}
}

func (c *crmStore) setPipelines(obj map[string]any) error {
	return atomicWriteFile(c.pipelinesPath(), marshalIndentJSON(obj))
}

func (c *crmStore) ensureDefaultPipelines() (map[string]any, error) {
	if _, err := os.Stat(c.pipelinesPath()); err != nil {
		if err := c.setPipelines(defaultPipelines()); err != nil {
			return nil, err
		}
	}
	return c.getPipelines(), nil
}

// --- segments -----------------------------------------------------------------

func (c *crmStore) segmentsPath() string { return filepath.Join(c.crmRoot, "segments.json") }

func (c *crmStore) getSegments() map[string]any {
	if m, err := readJSONFile(c.segmentsPath()); err == nil {
		return m
	}
	return map[string]any{"segments": []any{}}
}

func (c *crmStore) setSegment(seg map[string]any) (map[string]any, error) {
	data := c.getSegments()
	var segs []any
	for _, s := range mList(data, "segments") {
		if sm, ok := s.(map[string]any); ok && sm["id"] == seg["id"] {
			continue
		}
		segs = append(segs, s)
	}
	segs = append(segs, seg)
	data["segments"] = segs
	if err := atomicWriteFile(c.segmentsPath(), marshalIndentJSON(data)); err != nil {
		return nil, err
	}
	return seg, nil
}

func condsFromSpec(raw []any) ([]cond, error) {
	var out []cond
	for _, item := range raw {
		trip, ok := item.([]any)
		if !ok || len(trip) != 3 {
			return nil, storageErrf("bad where condition %v", item)
		}
		field, _ := trip[0].(string)
		op, _ := trip[1].(string)
		out = append(out, cond{Field: field, Op: op, Value: trip[2]})
	}
	return out, nil
}

func (c *crmStore) resolveSegment(segID string) ([]map[string]any, error) {
	var seg map[string]any
	for _, s := range mList(c.getSegments(), "segments") {
		if sm, ok := s.(map[string]any); ok && mStr(sm, "id") == segID {
			seg = sm
			break
		}
	}
	if seg == nil {
		return nil, storageErrf("segment %q not found", segID)
	}
	where, err := condsFromSpec(mList(seg, "where"))
	if err != nil {
		return nil, err
	}
	rows, err := c.a.query("contacts", where, nil, -1, 0)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, ct := range rows {
		if mStr(mMap(ct, "merge"), "status") == "merged" {
			continue
		}
		if mStr(ct, "lifecycle_stage") == "do_not_contact" {
			continue
		}
		if c.contactSuppressed(ct) {
			continue
		}
		out = append(out, ct)
	}
	return out, nil
}

func (c *crmStore) contactSuppressed(contact map[string]any) bool {
	for _, pair := range c.identityPairs(contact) {
		switch pair[0] {
		case "email":
			if c.isSuppressed(pair[1], "", nil) != nil {
				return true
			}
		case "phone":
			if c.isSuppressed("", pair[1], nil) != nil {
				return true
			}
		case "social":
			if c.isSuppressed("", "", []string{pair[1]}) != nil {
				return true
			}
		}
	}
	return false
}

// --- campaigns ----------------------------------------------------------------

func safeSlug(slug, what string) (string, error) {
	if slug == "" || strings.Contains(slug, "/") || strings.Contains(slug, "\\") || slug == "." || slug == ".." || strings.HasPrefix(slug, ".") {
		return "", storageErrf("unsafe %s %q", what, slug)
	}
	return slug, nil
}

func (c *crmStore) campaignDir(slug string) (string, error) {
	s, err := safeSlug(slug, "campaign slug")
	if err != nil {
		return "", err
	}
	return filepath.Join(c.clientDir, "campaigns", s), nil
}

func (c *crmStore) campaignConfigPath(slug string) (string, error) {
	d, err := c.campaignDir(slug)
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "campaign_config.json"), nil
}

func (c *crmStore) getCampaign(slug string) map[string]any {
	p, err := c.campaignConfigPath(slug)
	if err != nil {
		return nil
	}
	if m, err := readJSONFile(p); err == nil {
		return m
	}
	return nil
}

func (c *crmStore) listCampaigns() []map[string]any {
	root := filepath.Join(c.clientDir, "campaigns")
	var out []map[string]any
	entries, _ := os.ReadDir(root)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, slug := range names {
		if cfg := c.getCampaign(slug); cfg != nil {
			out = append(out, cfg)
		}
	}
	return out
}

var goalTypes = map[string]bool{"book_meeting": true, "get_reply": true, "direct_sale": true,
	"reactivation": true, "nurture_upsell": true, "event_invite": true}

func sortedGoalTypes() []string {
	out := make([]string, 0, len(goalTypes))
	for k := range goalTypes {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// A campaign's CHANNEL is declared at creation, because it decides what else the campaign
// needs: an email or messenger campaign targets a segment of contacts, a comment campaign
// targets a list of GROUPS. It also decides how the goal is written — the same offer needs a
// different pen for a cold email, a DM, and a public comment, which is why each channel gets
// its own campaign with its own goal rather than one campaign fanning out.
//
// `email_first` is the historical value and stays the default so existing configs keep
// working untouched.
var channelStrategies = map[string]bool{
	"email_first":    true, // email; the original and default
	"messenger":      true, // Messenger DM, executed by the collector after approval
	"comment":        true, // Facebook comments on group posts, executed by the collector
	"post":           true, // new posts INTO groups — highest exposure, lowest caps
	"friend_harvest": true, // Leads From Friends: walk seed profiles' friend lists into the CRM (harvest.go); sends nothing
	"zillow_harvest": true, // Leads From Zillow: walk agent-directory queries into the CRM (harvest_zillow.go); sends nothing
}

// normalizeZillowLocations validates + cleans pasted Zillow directory urls
// (one per line in the UI): zillow.com/professionals/... only, name/page
// params stripped (the walker owns them), deduped.
func normalizeZillowLocations(raw []any) ([]any, error) {
	var out []any
	seen := map[string]bool{}
	for _, v := range raw {
		sv := strings.TrimSpace(sprint(v))
		if sv == "" {
			continue
		}
		clean, err := zillowDirectoryURL(sv)
		if err != nil {
			return nil, storageErrf("zillow location: %v", err)
		}
		if seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	return out, nil
}

// normalizeZillowKeywords: comma- or newline-separated, lower-cased, deduped.
func normalizeZillowKeywords(raw any) []any {
	var parts []string
	switch v := raw.(type) {
	case string:
		parts = strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == '\n' || r == ';' })
	case []any:
		for _, x := range v {
			parts = append(parts, strings.FieldsFunc(sprint(x), func(r rune) bool { return r == ',' || r == '\n' || r == ';' })...)
		}
	}
	var out []any
	seen := map[string]bool{}
	for _, p := range parts {
		s := strings.ToLower(strings.TrimSpace(p))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// normalizeSeedProfiles cleans a pasted list of profile urls (one per line in
// the UI) into canonical store-form urls, de-duplicated by identity.
func normalizeSeedProfiles(raw []any) ([]any, error) {
	var out []any
	seen := map[string]bool{}
	for _, v := range raw {
		s := strings.TrimSpace(sprint(v))
		if s == "" {
			continue
		}
		clean, ok := canonicalStoreURL(s)
		if !ok {
			return nil, storageErrf("seed profile %q has no derivable identity (an opaque share link?) — paste the profile page url", s)
		}
		uid, _ := sourceUID(clean)
		if strings.HasPrefix(uid, "facebook.com/groups/") {
			return nil, storageErrf("seed %q is a GROUP, not a profile — Leads From Friends walks a person's friend list", s)
		}
		if seen[uid] {
			continue
		}
		seen[uid] = true
		out = append(out, clean)
	}
	return out, nil
}

// validateHarvestBlock keeps only known keys with sane ranges. Zero/absent =
// "use the system setting" (harvestConfigFrom does the fallback).
func validateHarvestBlock(h map[string]any) (map[string]any, error) {
	out := map[string]any{}
	intIn := func(key string, min, max int) error {
		if v, ok := h[key]; ok && v != nil {
			n := int(asFloat(v, -1))
			if n == 0 {
				return nil
			}
			if n < min || n > max {
				return storageErrf("harvest.%s must be %d..%d, got %v", key, min, max, v)
			}
			out[key] = n
		}
		return nil
	}
	if err := intIn("daily_budget", 1, 5000); err != nil {
		return nil, err
	}
	if err := intIn("per_collector_budget", 1, 2000); err != nil {
		return nil, err
	}
	if err := intIn("leg_pages", 1, 40); err != nil {
		return nil, err
	}
	for _, key := range []string{"quiet_from", "quiet_to"} {
		if v := strings.TrimSpace(mStr(h, key)); v != "" {
			if !runTimeRe.MatchString(v) {
				return nil, storageErrf("harvest.%s must be HH:MM, got %q", key, v)
			}
			out[key] = v
		}
	}
	if kw, ok := h["goal_keywords"].([]any); ok {
		var list []any
		for _, k := range kw {
			if s := strings.ToLower(strings.TrimSpace(sprint(k))); s != "" {
				list = append(list, s)
			}
		}
		if len(list) > 0 {
			out["goal_keywords"] = list
		}
	}
	return out, nil
}

// channelUsesGroups reports whether a campaign targets places (groups) rather than a segment
// of people. Comment and post campaigns both do.
func channelUsesGroups(channel string) bool {
	return channel == "comment" || channel == "post"
}

func sortedChannelStrategies() []string {
	out := make([]string, 0, len(channelStrategies))
	for k := range channelStrategies {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// normalizeGroupURLs cleans a group list into canonical, de-duplicated facebook group urls,
// preserving the operator's ordering. A group the agent cannot recognise is rejected rather
// than stored: a typo here means a comment campaign silently watches nothing.
func normalizeGroupURLs(raw []any) ([]any, error) {
	seen := map[string]bool{}
	out := []any{}
	for _, item := range raw {
		s := strings.TrimSpace(fmt.Sprint(item))
		if s == "" {
			continue
		}
		if !strings.Contains(strings.ToLower(s), "facebook.com/groups/") {
			return nil, storageErrf("not a facebook group url: %q", s)
		}
		s = strings.TrimRight(s, "/")
		key := strings.ToLower(s)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, s)
	}
	return out, nil
}

func (c *crmStore) createCampaign(slug string, config map[string]any) (map[string]any, error) {
	goal := mMap(config, "goal")
	gt := mStr(goal, "goal_type")
	if gt != "" && !goalTypes[gt] {
		return nil, storageErrf("goal_type %q not in %v", gt, sortedGoalTypes())
	}
	if gt == "" {
		gt = "get_reply"
	}
	// Validate the channel here rather than letting deepUpdate write anything through: a
	// mistyped channel would produce a campaign that looks configured and is never picked up
	// by any send engine.
	if cs := mStr(config, "channel_strategy"); cs != "" && !channelStrategies[cs] {
		return nil, storageErrf("channel_strategy %q not in %v", cs, sortedChannelStrategies())
	}
	// Refuse to overwrite a live campaign. This used to write straight through, which was
	// survivable while creation was CLI-only and deliberate; with a Create button on the
	// operator's screen, one repeated slug would silently wipe a running campaign's goal,
	// message bank and sendbox list. Editing is what `campaignUpdate` is for.
	if existing := c.getCampaign(slug); existing != nil {
		return nil, storageErrf("campaign %q already exists — edit it instead of recreating it", slug)
	}
	cfg := map[string]any{
		"schema_version": 1, "campaign_slug": slug,
		"goal": map[string]any{"goal_type": gt, "description": "", "objective": "", "offer": "",
			"value_proposition": "", "proof_points": []any{},
			"cta":           map[string]any{"type": "reply_yes", "text": ""},
			"success_event": map[string]any{"on": "reply_positive", "create_deal_stage": "new_reply"}},
		// `segment` targets people (email / messenger); `groups` targets places (comment).
		// Both fields always exist so a campaign can be inspected without knowing its channel.
		"audience": map[string]any{"segment": "", "groups": []any{}, "personalization": map[string]any{
			"required_hook_types": []any{}, "min_confidence": 0.7, "no_hook_fallback": "skip"}},
		"sequence": []any{
			map[string]any{"step": 1, "intent": "hook + offer, one CTA", "tracking": "plain_text"},
			map[string]any{"step": 2, "gap_days": 4, "intent": "deliver new value"},
			map[string]any{"step": 3, "gap_days": 5, "intent": "social proof"},
			map[string]any{"step": 4, "gap_days": 7, "intent": "breakup"}},
		"sendboxes": []any{}, "daily_quota": 40, "approval_mode": "manual_all",
		"channel_strategy":                          "email_first",
		"min_days_between_touches_across_campaigns": 7,
		"guardrails":                                map[string]any{"banned_claims": []any{"guarantees"}, "no_fake_re": true},
		"status":                                    "active",
	}
	deepUpdate(cfg, config)
	cfg["campaign_slug"] = slug
	// Leads-From-Friends: seed profiles and the harvest block go through the same
	// validators as campaignUpdate, so a create-with-JSON cannot smuggle a dirty
	// url or an out-of-range budget past what the UI edit path refuses.
	if mStr(cfg, "channel_strategy") == zillowChannel {
		// Same contract as friend_harvest: created PAUSED, inputs validated on create.
		if _, explicit := config["status"]; !explicit {
			cfg["status"] = "paused"
		}
		if raw, ok := cfg["zillow_locations"].([]any); ok {
			locs, err := normalizeZillowLocations(raw)
			if err != nil {
				return nil, err
			}
			cfg["zillow_locations"] = locs
		} else {
			cfg["zillow_locations"] = []any{}
		}
		cfg["zillow_keywords"] = normalizeZillowKeywords(cfg["zillow_keywords"])
		if hm, ok := cfg["harvest"].(map[string]any); ok {
			clean, err := validateHarvestBlock(hm)
			if err != nil {
				return nil, err
			}
			cfg["harvest"] = clean
		} else {
			cfg["harvest"] = map[string]any{}
		}
	}
	if mStr(cfg, "channel_strategy") == harvestChannel {
		// A harvest campaign touches Facebook on its own (no approval gate), so it
		// must never start the moment it is saved: the operator is still pasting
		// seeds and writing the goal. It is created PAUSED; "Start harvest" on the
		// campaign page (or `campaign update` status:active) is the deliberate on.
		if _, explicit := config["status"]; !explicit {
			cfg["status"] = "paused"
		}
		if raw, ok := cfg["seed_profiles"].([]any); ok {
			seeds, err := normalizeSeedProfiles(raw)
			if err != nil {
				return nil, err
			}
			cfg["seed_profiles"] = seeds
		} else {
			cfg["seed_profiles"] = []any{}
		}
		if hm, ok := cfg["harvest"].(map[string]any); ok {
			clean, err := validateHarvestBlock(hm)
			if err != nil {
				return nil, err
			}
			cfg["harvest"] = clean
		} else {
			cfg["harvest"] = map[string]any{}
		}
	}
	dir, err := c.campaignDir(slug)
	if err != nil {
		return nil, err
	}
	for _, sub := range []string{"queue/enriched", "outbox/pending_approval", "outbox/approved", "history"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(sub)), 0o755); err != nil {
			return nil, err
		}
	}
	p, _ := c.campaignConfigPath(slug)
	if err := atomicWriteFile(p, marshalIndentJSON(cfg)); err != nil {
		return nil, err
	}
	return cfg, nil
}

// --- enrich queue (JIT buffer) -------------------------------------------------

func (c *crmStore) enrichQueuePath(slug string) string {
	d, _ := c.campaignDir(slug)
	return filepath.Join(d, "queue", "enrich_queue.jsonl")
}

func (c *crmStore) queuedOrSentLeads(slug string) map[string]bool {
	out := map[string]bool{}
	for _, row := range readJSONLines(c.enrichQueuePath(slug)) {
		if lid := mStr(row, "lead_id"); lid != "" {
			out[c.resolve(lid)] = true
		}
	}
	for _, p := range c.allSentLogs(slug) {
		for _, row := range readJSONLines(p) {
			if lid := mStr(row, "lead_id"); lid != "" {
				out[c.resolve(lid)] = true
			}
		}
	}
	return out
}

func (c *crmStore) allSentLogs(onlyCampaign string) []string {
	root := filepath.Join(c.clientDir, "campaigns")
	var files []string
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		camp := e.Name()
		if onlyCampaign != "" && camp != onlyCampaign {
			continue
		}
		base := filepath.Join(root, camp, "sent")
		months, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		names := make([]string, 0, len(months))
		for _, m := range months {
			names = append(names, m.Name())
		}
		sort.Strings(names)
		for _, m := range names {
			fp := filepath.Join(base, m, "sent_log.jsonl")
			if st, err := os.Stat(fp); err == nil && !st.IsDir() {
				files = append(files, fp)
			}
		}
	}
	return files
}

func (c *crmStore) lastTouchOtherCampaign(leadID, thisCampaign string) string {
	target := c.resolve(leadID)
	latest := ""
	for _, p := range c.allSentLogs("") {
		for _, r := range readJSONLines(p) {
			if c.resolve(mStr(r, "lead_id")) == target && mStr(r, "campaign") != thisCampaign {
				if sa := mStr(r, "sent_at"); sa > latest {
					latest = sa
				}
			}
		}
	}
	return latest
}

func (c *crmStore) queueCampaign(slug string, limit int) (map[string]any, error) {
	cfg := c.getCampaign(slug)
	if cfg == nil {
		return nil, storageErrf("campaign %q not found", slug)
	}
	if mStr(cfg, "status") == "paused" {
		// a paused campaign is a clean no-op for the daily run, not an error
		return map[string]any{"campaign": slug, "queued": 0,
			"skipped": map[string]any{"campaign_paused": 1},
			"segment": mStr(mMap(cfg, "audience"), "segment"), "paused": true}, nil
	}
	segID := mStr(mMap(cfg, "audience"), "segment")
	if segID == "" {
		return nil, storageErrf("campaign %q has no audience.segment", slug)
	}
	minDays := mInt(cfg, "min_days_between_touches_across_campaigns", 7)
	cutoff := isoDaysAgo(minDays)
	candidates, err := c.resolveSegment(segID)
	if err != nil {
		return nil, err
	}
	added := 0
	skipped := map[string]any{"already_in_campaign": 0, "recently_touched_elsewhere": 0,
		"in_active_sequence": 0, "no_email": 0, "duplicate_suspected": 0}
	bump := func(k string) { skipped[k] = skipped[k].(int) + 1 }
	qp := c.enrichQueuePath(slug)
	if err := os.MkdirAll(filepath.Dir(qp), 0o755); err != nil {
		return nil, err
	}
	emailFirst := true
	if cs, ok := cfg["channel_strategy"].(string); ok {
		emailFirst = cs == "email_first"
	}
	slugSafe, _ := safeSlug(slug, "campaign slug")
	unlock, err := c.a.lock("queue_" + slugSafe)
	if err != nil {
		return nil, err
	}
	defer unlock()
	already := c.queuedOrSentLeads(slug)
	qf, err := os.OpenFile(qp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	defer qf.Close()
	for _, ct := range candidates {
		if added >= limit {
			break
		}
		leadID := c.resolve(mStr(ct, "id"))
		if already[leadID] {
			bump("already_in_campaign")
			continue
		}
		if len(mList(ct, "duplicate_suspects")) > 0 {
			// possibly the same human as another record — emailing both would
			// double-touch one person; operator resolves via merge/unsuspect
			bump("duplicate_suspected")
			continue
		}
		if emailFirst {
			hasEmail := false
			for _, e := range mapsOf(mList(mMap(ct, "identities"), "emails")) {
				if validEmail(mStr(e, "address")) {
					hasEmail = true
					break
				}
			}
			if !hasEmail {
				nf := mStr(mMap(ct, "enrichment"), "email_not_found_at")
				if nf != "" && nf >= isoDaysAgo(negRetryDays) {
					bump("no_email")
					continue
				}
			}
		}
		if mStr(ct, "sequence_state") == "frozen" {
			bump("in_active_sequence")
			continue
		}
		if last := c.lastTouchOtherCampaign(leadID, slug); last != "" && last >= cutoff {
			bump("recently_touched_elsewhere")
			continue
		}
		line := marshalLineJSON(map[string]any{"lead_id": leadID, "campaign": slug,
			"queued_at": nowISO(), "status": "queued", "step": 1})
		if _, err := qf.WriteString(line + "\n"); err != nil {
			return nil, err
		}
		already[leadID] = true
		added++
	}
	return map[string]any{"campaign": slug, "queued": added, "skipped": skipped, "segment": segID}, nil
}

// --- validation / migration ----------------------------------------------------

func (c *crmStore) validate(rebuildIndex bool) (map[string]any, error) {
	report := map[string]any{"contacts": 0, "problems": []any{}, "index_rebuilt": false, "identities_indexed": 0}
	problems := []any{}
	addProblem := func(s string) { problems = append(problems, s) }
	req := []string{"id", "schema_version", "created_at", "updated_at"}
	contactsDir := filepath.Join(c.crmRoot, "contacts")
	var contacts []map[string]any
	nContacts := 0
	entries, _ := os.ReadDir(contactsDir)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		rec, err := readJSONFile(filepath.Join(contactsDir, name))
		if err != nil {
			addProblem(fmt.Sprintf("%s: unreadable (%s)", name, pyErrClass(err)))
			continue
		}
		nContacts++
		for _, k := range req {
			if isEmptyValue(rec[k]) {
				addProblem(fmt.Sprintf("%s: missing/empty %s", name, k))
			}
		}
		if id := mStr(rec, "id"); id != "" && id != strings.TrimSuffix(name, ".json") {
			addProblem(fmt.Sprintf("%s: id %q != filename", name, id))
		}
		contacts = append(contacts, rec)
	}
	for _, coll := range []string{"accounts", "deals"} {
		d := filepath.Join(c.crmRoot, coll)
		es, _ := os.ReadDir(d)
		ns := make([]string, 0, len(es))
		for _, e := range es {
			ns = append(ns, e.Name())
		}
		sort.Strings(ns)
		for _, name := range ns {
			if strings.HasSuffix(name, ".json") {
				if _, err := readJSONFile(filepath.Join(d, name)); err != nil {
					addProblem(fmt.Sprintf("%s/%s: unreadable (%s)", coll, name, pyErrClass(err)))
				}
			}
		}
	}
	report["contacts"] = nContacts
	if rebuildIndex {
		idxPath := filepath.Join(c.crmRoot, "contact_identities.jsonl")
		var sb strings.Builder
		seq, n := 0, 0
		for _, rec := range contacts {
			if mStr(mMap(rec, "merge"), "status") == "merged" {
				continue
			}
			for _, pair := range c.identityPairs(rec) {
				seq++
				sb.WriteString(marshalLineJSON(map[string]any{"seq": seq, "ts": nowISO(),
					"kind": pair[0], "value": pair[1], "contact_id": mStr(rec, "id"), "removed": false}) + "\n")
				n++
			}
		}
		tmp := idxPath + ".rebuild.tmp"
		if err := os.WriteFile(tmp, []byte(sb.String()), 0o644); err != nil {
			return nil, err
		}
		if err := os.Rename(tmp, idxPath); err != nil {
			return nil, err
		}
		c.a.identityCacheSet = false
		report["index_rebuilt"] = true
		report["identities_indexed"] = n
	}
	report["problems"] = problems
	return report, nil
}

// pyErrClass maps a Go read/parse error to the Python exception class name the
// validate report embeds (tests match on these strings loosely).
func pyErrClass(err error) string {
	if _, ok := err.(*json.SyntaxError); ok {
		return "ValueError"
	}
	if _, ok := err.(*json.UnmarshalTypeError); ok {
		return "ValueError"
	}
	if os.IsNotExist(err) || os.IsPermission(err) {
		return "OSError"
	}
	if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unexpected end of JSON") {
		return "ValueError"
	}
	return "OSError"
}

// isEmptyValue mirrors Python falsiness for the fields validate checks.
func isEmptyValue(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case bool:
		return !x
	case float64:
		return x == 0
	}
	return false
}

// --- enrichment ----------------------------------------------------------------

const (
	// Operator decision 2026-07-27: identity and hooks stay usable for a YEAR.
	// The 90/10 pair dragged every contact back through full enrichment ~4x and
	// hook-refresh ~36x a year, which contradicted the enrich-once doctrine and
	// burned collector budget on leads that had nothing new to say. The accepted
	// trade: a person who changed jobs is caught later, at SEND time, by the
	// bounce path (hard_bounce -> suppress + freeze), not proactively.
	identityTTLDays = 360
	hookTTLDays     = 360
	negRetryDays    = 30
	// A hook older than this cannot carry a `high` band ON ITS OWN. Was 60 days,
	// which skipped real leads over a one-day miss; a substantive signal from
	// within the past year is still worth writing about in cold outreach.
	// Hooks that cite a URL the OPERATOR supplied are exempt entirely (below).
	hookRecencyDays = 360
)

// contactHasUsableEmail: at least one syntactically valid address on file.
func contactHasUsableEmail(ct map[string]any) bool {
	for _, e := range mapsOf(mList(mMap(ct, "identities"), "emails")) {
		if validEmail(mStr(e, "address")) {
			return true
		}
	}
	return false
}

// dossierLookedBeyondFacebook reports whether this dossier carries ANY evidence
// URL from outside Facebook. It is the verifiable proxy for "the email hunt
// actually left the Facebook pass": a run that only read seeds/profiles on
// facebook.com cannot have consulted a website, roster, directory or search
// result, so it has not earned the right to declare the address unfindable.
// Deliberately checks the DATA the dossier carries, not a checklist the agent
// fills in — a self-reported ladder is exactly what drifts.
// fbContactSurfaces are the Facebook URLs that actually expose an address. The
// About TAB alone is not one of them: on most business profiles About renders a
// chooser and the address lives in a sub-page (Contact info / Intro / Basic info
// / Links), each a distinct URL. Checking only "/about" would both miss those and
// reject a run that correctly visited them.
var fbContactSurfaces = []string{
	"directory_contact_info", "directory_intro", "directory_basic_info", "directory_links",
	"sk=about_contact_and_basic_info", "sk=about_details", "sk=about_overview", "sk=about",
	"/about",
}

// --- the contact's NAME: display vs greetable ---------------------------------
//
// The dossier schema had no field for the name enrichment discovers, so the
// header processor improvised: it wrote the observed name into
// enrichment.context.header_name AND identity.current_company (identical value
// in 182 of 182 live contacts, which also means a person's name was being
// stored as their COMPANY). enrichWrite never promoted either to
// contact.name.full, so the UI fell back to the ShortID and 360 of 404 live
// contacts displayed as "c_...".
//
// Fixing that is a promotion, but a blind promotion is worse than the bug: 34
// of those 182 values are not names at all (UI text "Post 2"/"Facebook", phone
// numbers, emoji taglines, marketing sentences), and a wrong name goes into the
// GREETING of a real email. So two questions, deliberately separate:
//
//   displayNameOK  — fit to show as this contact's name in the CRM?
//   greetableName  — safe for a writer to put after "Chào"/"Hi"?
//
// They differ on the middle class Facebook pages love: "Jessie Nguyen -
// Mortgage Advisor", "Cuộc Sống Mỹ - Mai Vu". Perfectly good display names, and
// greeting with one whole would read as a mail merge. Measured on the live pool:
// 148 displayable, 114 of those greetable.

var (
	uiTextNameRe     = regexp.MustCompile(`(?i)^\s*(post|reel|video|photos?|stor(y|ies)|facebook|meta|new message|messages?|message|home|follow(ing)?|see more|xem thêm|tin nhắn|trang chủ|theo dõi|about|intro)\b\s*\d*\s*$`)
	contactishNameRe = regexp.MustCompile(`(?i)(zalo|phone|\+?\d[\d.\s-]{7,})`)
	sentenceyNameRe  = regexp.MustCompile(`(?i)[.!?…]\s|[.!?…]$|\bhelping\b|\bchia sẻ\b|\bgiúp\b|\btôi đã\b`)
	titleishNameRe   = regexp.MustCompile(`(?i)\b(realtor|broker|nmls|dre|llc|inc|co\.|lpa|law firm|services|mortgage|advisor|expert|group|team)\b`)
	compositeNameRe  = regexp.MustCompile(`[-–—:|,]`)
)

func nameHasEmoji(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.So, unicode.Sk) {
			return true
		}
	}
	return false
}

// displayNameOK reports whether s is fit to be a contact's displayed name, and
// why not when it is not. Thresholds calibrated against the live pool, so a real
// business name ("Catholic MTA USA – Immigration Services") passes while a bio
// ("Helping Families Build Wealth Through Real Estate, Retirement and ...")
// does not.
func displayNameOK(s string) (bool, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return false, "empty"
	}
	if uiTextNameRe.MatchString(s) {
		return false, "page-chrome text, not a name"
	}
	if contactishNameRe.MatchString(s) {
		return false, "a phone/Zalo number, not a name"
	}
	if len([]rune(s)) > 60 {
		return false, "longer than 60 characters — a bio or tagline, not a name"
	}
	if len(strings.Fields(s)) > 8 {
		return false, "more than 8 words — a tagline, not a name"
	}
	if sentenceyNameRe.MatchString(s) {
		return false, "reads as a marketing sentence, not a name"
	}
	if nameHasEmoji(s) {
		return false, "emoji-decorated tagline, not a name"
	}
	letters, digits := 0, 0
	for _, r := range s {
		switch {
		case unicode.IsLetter(r):
			letters++
		case unicode.IsDigit(r):
			digits++
		}
	}
	if letters < 2 {
		return false, "no letters"
	}
	if digits*2 >= letters {
		return false, "mostly digits"
	}
	return true, ""
}

// greetableName reports whether a writer may address the reader with s as-is.
// Stricter than display: no composites (the person is one half of them), no
// role/licence words, at most four words.
func greetableName(s string) bool {
	if ok, _ := displayNameOK(s); !ok {
		return false
	}
	s = strings.TrimSpace(s)
	if compositeNameRe.MatchString(s) || titleishNameRe.MatchString(s) {
		return false
	}
	return len(strings.Fields(s)) <= 4
}

// seedResolutionState reads the seed's own account of WHY it is unresolved.
//
// The state machine could not tell a lead nobody can resolve from a collector
// job that hiccuped. On the live batch that mattered: 134 of 136 unresolved reel
// seeds came back with a Messenger overlay in the DOM instead of the owner link.
// The owner filter correctly refused that data (it had already prevented one
// wrong merge), but the run then treated all 136 as ordinary "unresolved",
// drafted the leads that were ready, and reported. Infrastructure noise looked
// exactly like an exhausted search.
//
// resolution.state ∈ resolved | retryable | exhausted. Written by the agent that
// ran the job, because only it knows what came back; enforced here.
func seedResolutionState(sd map[string]any) string {
	if st := strings.ToLower(strings.TrimSpace(mStr(mMap(sd, "resolution"), "state"))); st != "" {
		return st
	}
	if strings.EqualFold(mStr(sd, "status"), "resolved") {
		return "resolved"
	}
	return "" // unclassified: an older record, or a job that never reported back
}

// contactRetryableSeeds returns the seeds whose owner lookup failed for a reason
// that a retry can fix.
func contactRetryableSeeds(ct map[string]any) []string {
	var out []string
	for _, sd := range mapsOf(mList(mMap(ct, "identities"), "seeds")) {
		if seedResolutionState(sd) == "retryable" {
			if u := mStr(sd, "url"); u != "" {
				out = append(out, u)
			}
		}
	}
	return out
}

// contentSeedKinds are the seed kinds that ARE hook material. A profile or an
// email seed identifies a person; a reel/post/video IS a story about them, and
// when the operator saved one they were choosing the reason to make contact.
var contentSeedKinds = map[string]bool{
	"reel": true, "post": true, "video": true, "watch": true, "blog": true, "content": true,
}

// normSeedURL: a hook cites the seed by URL; compare them forgivingly.
func normSeedURL(u string) string {
	return strings.TrimRight(strings.ToLower(strings.TrimSpace(u)), "/")
}

// unharvestedSeedHooks: which of this contact's curated content seeds does no
// hook cite? Those are not "no hook found" — they are hook material nobody read.
func unharvestedSeedHooks(ct map[string]any, hooks []any) []string {
	cited := map[string]bool{}
	for _, h := range mapsOf(hooks) {
		if u := mStr(h, "evidence_url"); u != "" {
			cited[normSeedURL(u)] = true
		}
	}
	var out []string
	for _, sd := range mapsOf(mList(mMap(ct, "identities"), "seeds")) {
		u := mStr(sd, "url")
		if u == "" || !contentSeedKinds[strings.ToLower(mStr(sd, "kind"))] {
			continue
		}
		if !cited[normSeedURL(u)] {
			out = append(out, u)
		}
	}
	return out
}

// withEmailDiscovery folds the collector's contacts record into channels_found.
//
// The playbook has the agent copy that record into the dossier as
// `email_discovery`, VERBATIM. It carries the address the dig found — but the
// store only ever read `email_discovery.checked` (the audit trail the
// mark_email_not_found gate reads), so an agent doing exactly what it was told
// had the address dropped in silence: nothing persisted, `problems` empty, no
// error to notice. channels_found.emails stays the canonical home; this only
// makes literal compliance work. Copies rather than mutates: the dossier is
// snapshotted to disk verbatim and must stay the agent's own words.
func withEmailDiscovery(found, ed map[string]any) map[string]any {
	if len(ed) == 0 {
		return found
	}
	out := map[string]any{}
	for k, v := range found {
		out[k] = v
	}
	emails := append([]any{}, mList(found, "emails")...)
	seen := map[string]bool{}
	for _, e := range emails {
		seen[normalizeEmail(fmt.Sprint(e))] = true
	}
	for _, e := range mList(ed, "emails") {
		s, ok := e.(string)
		if !ok || s == "" || seen[normalizeEmail(s)] {
			continue
		}
		seen[normalizeEmail(s)] = true
		emails = append(emails, s)
	}
	if len(emails) > 0 {
		out["emails"] = emails
	}
	// the site the dig turned up is the website hop's input — keep it as an
	// identity too, but never overwrite a website the dossier named itself.
	if pm, ok := out["profiles"].(map[string]any); !ok || mStr(pm, "website") == "" {
		for _, w := range mList(ed, "websites") {
			s, ok := w.(string)
			if !ok || s == "" {
				continue
			}
			np := map[string]any{}
			if pm != nil {
				for k, v := range pm {
					np[k] = v
				}
			}
			np["website"] = s
			out["profiles"] = np
			break
		}
	}
	return out
}

// dossierContactsLadderRan: the collector's `fb.profile.contacts` capability walks
// the whole ladder inside ONE job, so its record's URL is the profile BASE — no
// evidence URL ever carries a directory_* slug even on a perfect run. What it does
// carry is `checked`: the surfaces that actually RENDERED, written by the extension,
// not by the agent. Copy it into the dossier as `email_discovery.checked` and this
// counts as having read the contact surfaces.
//
// Passing needs more than "current_page": a bare page read is what the old failing
// passes already did. Either a directory_* sub-tab rendered, or the ladder entered
// About and found no sub-tab to click — which is the honest outcome for most personal
// profiles (12 of 13 verified by hand), and refusing those would park them forever.
func dossierContactsLadderRan(dossier map[string]any) bool {
	ed := mMap(dossier, "email_discovery")
	if ed == nil {
		return false
	}
	for _, v := range mList(ed, "checked") {
		s := strings.ToLower(fmt.Sprint(v))
		if s == "about" {
			return true
		}
		for _, t := range fbContactSurfaces {
			if strings.Contains(s, t) {
				return true
			}
		}
	}
	return false
}

// dossierReadAboutTab: did any evidence URL point at a Facebook surface that can
// actually show an address? Verifiable from the data, unlike a self-reported
// "I checked the profile".
func dossierReadAboutTab(dossier, ident map[string]any) bool {
	if dossierContactsLadderRan(dossier) {
		return true
	}
	for _, u := range dossierEvidenceURLs(dossier, ident) {
		l := strings.ToLower(u)
		if !strings.Contains(l, "facebook.com") {
			continue
		}
		for _, s := range fbContactSurfaces {
			if strings.Contains(l, s) {
				return true
			}
		}
	}
	return false
}

// dossierEvidenceURLs collects every URL this dossier cites.
func dossierEvidenceURLs(dossier, ident map[string]any) []string {
	var urls []string
	for _, h := range mapsOf(mList(dossier, "hooks")) {
		urls = append(urls, mStr(h, "evidence_url"))
	}
	for _, e := range mapsOf(mList(ident, "evidence")) {
		urls = append(urls, mStr(e, "url"))
	}
	found := mMap(ident, "channels_found")
	switch pv := found["profiles"].(type) {
	case map[string]any:
		for _, u := range pv {
			urls = append(urls, fmt.Sprint(u))
		}
	case []any:
		for _, u := range pv {
			urls = append(urls, fmt.Sprint(u))
		}
	}
	return append(urls, mStr(ident, "website"), mStr(found, "website"))
}

func dossierLookedBeyondFacebook(dossier, ident map[string]any) bool {
	fbHost := func(u string) bool {
		l := strings.ToLower(u)
		return strings.Contains(l, "facebook.com") || strings.Contains(l, "fb.com") ||
			strings.Contains(l, "fb.watch") || strings.Contains(l, "fbcdn.net")
	}
	var urls []string
	for _, h := range mapsOf(mList(dossier, "hooks")) {
		urls = append(urls, mStr(h, "evidence_url"))
	}
	for _, e := range mapsOf(mList(ident, "evidence")) {
		urls = append(urls, mStr(e, "url"))
	}
	found := mMap(ident, "channels_found")
	switch pv := found["profiles"].(type) {
	case map[string]any:
		for _, u := range pv {
			urls = append(urls, fmt.Sprint(u))
		}
	case []any:
		for _, u := range pv {
			urls = append(urls, fmt.Sprint(u))
		}
	}
	urls = append(urls, mStr(ident, "website"), mStr(found, "website"))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" && !fbHost(u) && (strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")) {
			return true
		}
	}
	return false
}

func (c *crmStore) enrichStatus(contactID, now string) map[string]any {
	if now == "" {
		now = nowISO()
	}
	ct := c.getContact(contactID)
	if ct == nil {
		return map[string]any{"needs": "skip", "reason": "contact_not_found"}
	}
	// Resolution ladder (DESIGN §7.1/§9): a contact with NO anchor (no email/
	// phone/profile/website) must first be traced back to its origin profile —
	// from its content seeds when it has any, else from the name fragment.
	if !hasContactAnchor(ct) {
		// A collector failure that a retry can fix is NOT an exhausted search, and
		// must not hide inside seed_unresolved — that reads as "the hunt is done
		// and came up empty", which is how 134 overlay-contaminated jobs got
		// counted as finished work.
		if urls := contactRetryableSeeds(ct); len(urls) > 0 {
			return map[string]any{"needs": "enrich", "reason": "seed_retryable_failure", "seed_urls": urls}
		}
		for _, sd := range mapsOf(mList(mMap(ct, "identities"), "seeds")) {
			if mStr(sd, "status") != "resolved" {
				return map[string]any{"needs": "enrich", "reason": "seed_unresolved"}
			}
		}
		if strings.TrimSpace(mStr(mMap(ct, "name"), "full")) != "" {
			return map[string]any{"needs": "enrich", "reason": "name_only_fragment"}
		}
	}
	en := mMap(ct, "enrichment")
	ident := mMap(en, "identity")
	identFresh := mStr(ident, "enriched_at") != "" && mStr(ident, "enriched_at") >= isoDaysAgoFrom(identityTTLDays, now)
	// A dossier with ZERO hooks is never "fresh" no matter how recently we looked:
	// the long hook TTL is there so a lead that HAS material is not re-hunted, not
	// so a lead we found nothing on is parked for a year. Empty-handed leads are
	// governed by the no_verifiable_hook negative cache below (30 days) instead.
	hooksFresh := len(mList(en, "hooks")) > 0 &&
		mStr(en, "hooks_refreshed_at") != "" && mStr(en, "hooks_refreshed_at") >= isoDaysAgoFrom(hookTTLDays, now)
	nf := mStr(en, "email_not_found_at")
	nh := mStr(en, "no_verifiable_hook_at")
	if mStr(ident, "still_active") == "inactive" && identFresh {
		return map[string]any{"needs": "skip", "reason": "known_inactive"}
	}
	if !identFresh {
		return map[string]any{"needs": "enrich", "reason": "identity_stale_or_missing"}
	}
	if nf != "" && nf >= isoDaysAgoFrom(negRetryDays, now) {
		return map[string]any{"needs": "skip", "reason": "email_not_found_recent"}
	}
	// `no_verifiable_hook_at` was WRITTEN by enrich write but never read here, so a
	// lead the springboard had already exhausted came back as "hooks_stale" every
	// hookTTLDays (10) forever — re-burning collector calls and tokens on the most
	// hopeless leads. It is a negative cache like email_not_found and honors the
	// same 30-day window: don't re-hunt a hook we already failed to find.
	// A curated content seed that no hook cites is UNFINISHED WORK, and it must
	// not hide under `hooks_stale` — that reads like a routine refresh, so a run
	// counted these leads as "skipped" and moved on. The lead the operator
	// hand-picked is precisely the one that must never be quietly dropped.
	// Deliberately ahead of the no-hook negative cache: leads parked by an older
	// build that had no such check are exactly the ones worth re-opening.
	if len(mList(en, "hooks")) == 0 {
		if u := unharvestedSeedHooks(ct, nil); len(u) > 0 {
			return map[string]any{"needs": "enrich", "reason": "seed_hook_unharvested", "seed_urls": u}
		}
	}
	if nh != "" && nh >= isoDaysAgoFrom(negRetryDays, now) {
		return map[string]any{"needs": "skip", "reason": "no_verifiable_hook_recent"}
	}
	if !hooksFresh {
		return map[string]any{"needs": "refresh", "reason": "hooks_stale"}
	}
	// Hooks in hand but still no address, and nobody has declared the search
	// exhausted (`email_not_found_at` empty): the email hunt is UNFINISHED WORK,
	// not a dead end. It stays in the due queue under its own reason so the run
	// reports "research pending" instead of quietly counting the lead as skipped.
	// This is the exact case that produced "99 hooks, 0 emails, 100 skipped".
	if !contactHasUsableEmail(ct) {
		return map[string]any{"needs": "enrich", "reason": "email_discovery",
			"confidence_band": en["confidence_band"]}
	}
	return map[string]any{"needs": "skip", "reason": "dossier_fresh", "confidence_band": en["confidence_band"]}
}

func (c *crmStore) enrichDue(campaignSlug string, limit int, now string) []map[string]any {
	out := []map[string]any{}
	for _, row := range readJSONLines(c.enrichQueuePath(campaignSlug)) {
		st := c.enrichStatus(mStr(row, "lead_id"), now)
		needs := mStr(st, "needs")
		if needs == "enrich" || needs == "refresh" {
			out = append(out, map[string]any{"lead_id": row["lead_id"], "needs": needs, "reason": st["reason"]})
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

var evidenceURLRe = regexp.MustCompile(`(?i)^https?://[^\s/]+`)

func validEvidenceURL(url any) bool {
	s, ok := url.(string)
	return ok && evidenceURLRe.MatchString(strings.TrimSpace(s))
}

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func validEmail(address string) bool {
	return emailRe.MatchString(strings.TrimSpace(address))
}

func (c *crmStore) enrichWrite(contactID string, dossier map[string]any, campaignSlug string) (map[string]any, error) {
	leadID := c.resolve(contactID)
	// Many list rows are content fragments (reels/posts) of ONE person: when
	// this dossier's found channels already belong to other contacts,
	// consolidate BEFORE writing so the dossier lands on the single survivor
	// (safe pairs auto-merge; conflicting pairs are flagged + withheld).
	leadID2, consolidated, dupSuspects, skipIdent, err := c.consolidateOnDiscovery(
		leadID, mMap(mMap(dossier, "identity"), "channels_found"))
	if err != nil {
		return nil, err
	}
	leadID = leadID2
	now := nowISO()
	problems := []any{}
	var usableHooks []map[string]any
	brief := mMap(dossier, "writing_brief")
	doNotMention := append([]any{}, mList(brief, "do_not_mention")...)
	for _, h := range mapsOf(mList(dossier, "hooks")) {
		sens := mStr(mMap(h, "analysis"), "sensitivity")
		if sens == "" {
			sens = "public_business"
		}
		if sens == "personal" {
			if s := mStr(h, "summary"); s != "" {
				doNotMention = append(doNotMention, s)
			}
			continue
		}
		if !validEvidenceURL(h["evidence_url"]) {
			problems = append(problems, fmt.Sprintf("hook %s dropped: evidence_url missing or not a valid http(s) URL", pyRepr(hookType(h))))
			continue
		}
		hcv := h["confidence"]
		if hcv == nil {
			hcv = 0.0
		}
		hc, hcOK := floatOrNil(hcv)
		if !hcOK {
			problems = append(problems, fmt.Sprintf("hook %s: non-numeric confidence, treated as 0.0", pyRepr(hookType(h))))
			hc = 0.0
		}
		if mStr(h, "observed_date") == "" {
			problems = append(problems, fmt.Sprintf("hook %s: no observed_date (recency unverified)", pyRepr(hookType(h))))
		}
		usedIn := mList(h, "used_in")
		if usedIn == nil {
			usedIn = []any{}
		}
		usableHooks = append(usableHooks, map[string]any{"type": h["type"], "summary": mStr(h, "summary"),
			"evidence_url": strings.TrimSpace(mStr(h, "evidence_url")), "observed_date": mStr(h, "observed_date"),
			"confidence": hc, "sensitivity": sens, "used_in": usedIn})
	}
	ident := mMap(dossier, "identity")
	confV := brief["personalization_confidence"]
	if confV == nil {
		confV = 0.0
	}
	conf, confOK := floatOrNil(confV)
	if !confOK {
		problems = append(problems, "personalization_confidence non-numeric, treated as 0.0")
		conf = 0.0
	}
	band := "fallback"
	if conf >= 0.7 {
		band = "high"
	} else if conf >= 0.4 {
		band = "review_carefully"
	}
	prevContact := c.getContact(leadID)
	prevEn := mMap(prevContact, "enrichment")

	if campaignSlug != "" {
		cd, err := c.campaignDir(campaignSlug)
		if err != nil {
			return nil, err
		}
		d := filepath.Join(cd, "queue", "enriched", todayStr(now))
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
		full := map[string]any{}
		for k, v := range dossier {
			full[k] = v
		}
		full["lead_id"] = leadID
		full["enriched_at"] = now
		if err := atomicWriteFile(filepath.Join(d, leadID+".json"), marshalIndentJSON(full)); err != nil {
			return nil, err
		}
	}
	prevIdent := mMap(prevEn, "identity")
	// The industry gate sits HERE, before the merge, and DELETES a bad value rather than rejecting
	// the write. Two reasons. Rejecting would throw away the hooks, the brief and the channels that
	// came in the same dossier because one field was mistyped — an expensive punishment for a
	// vocabulary error. And deleting lets the loop below fall back to prevIdent, so a bad submission
	// cannot destroy a good stored value; a hard error would leave the old value in place too, but
	// only by abandoning everything else in the same call.
	//
	// It is reported in `problems`, which the enrich playbook already tells the agent to read. A
	// silently dropped field is the failure mode this file has been bitten by before: the address
	// that "had the address dropped in silence: nothing persisted, problems empty, no ..." note a
	// few hundred lines up is the same lesson.
	if raw, ok := ident["industry"]; ok {
		v := strings.TrimSpace(fmt.Sprint(raw))
		if raw == nil || v == "" {
			// Empty is a legitimate answer — the playbook says thin signals must leave it empty
			// rather than guess, because a wrong label mis-targets every later message invisibly.
			delete(ident, "industry")
		} else if !validLeadIndustry(v) {
			problems = append(problems, fmt.Sprintf("industry %s not in the lead industry dictionary; dropped (value must match lead_industries.json verbatim, including case and '&'). Allowed: %v", pyRepr(v), sortedLeadIndustries()))
			delete(ident, "industry")
			delete(ident, "industry_confidence")
			delete(ident, "industry_reason")
			delete(ident, "industry_signals")
		}
	}
	hasIdentity := len(ident) > 0
	mergedIdent := map[string]any{}
	// industry* travels with identity: the key list here is an ALLOWLIST, so a field missing from it
	// is silently discarded no matter what the caller sent. That is how enrichment fields have gone
	// missing before — the write succeeds, the field is simply not there afterwards.
	for _, k := range []string{"still_active", "current_company", "role", "profiles", "evidence",
		"industry", "industry_confidence", "industry_reason", "industry_signals"} {
		if v, ok := ident[k]; ok && v != nil {
			mergedIdent[k] = v
		} else if pv, ok := prevIdent[k]; ok {
			mergedIdent[k] = pv
		} else {
			mergedIdent[k] = nil
		}
	}
	if hasIdentity {
		mergedIdent["enriched_at"] = now
	} else if pe := mStr(prevIdent, "enriched_at"); pe != "" {
		mergedIdent["enriched_at"] = pe
	} else {
		mergedIdent["enriched_at"] = now
	}
	retired := map[string]bool{}
	for _, u := range mList(dossier, "retired_hooks") {
		if s := strings.TrimSpace(fmt.Sprint(u)); s != "" && u != nil {
			retired[s] = true
		}
	}
	// merge key = evidence_url; resubmit wins on content but unions used_in; prior hooks survive
	hooksByURL := map[string]map[string]any{}
	var urlOrder []string
	for _, h := range mapsOf(mList(prevEn, "hooks")) {
		u := mStr(h, "evidence_url")
		if u != "" && !retired[u] {
			cp := map[string]any{}
			for k, v := range h {
				cp[k] = v
			}
			if _, seen := hooksByURL[u]; !seen {
				urlOrder = append(urlOrder, u)
			}
			hooksByURL[u] = cp
		}
	}
	for _, h := range usableHooks {
		u := mStr(h, "evidence_url")
		if retired[u] {
			continue
		}
		if prev, ok := hooksByURL[u]; ok {
			union := map[string]bool{}
			for _, x := range mList(prev, "used_in") {
				if s, ok := x.(string); ok {
					union[s] = true
				}
			}
			for _, x := range mList(h, "used_in") {
				if s, ok := x.(string); ok {
					union[s] = true
				}
			}
			keys := make([]string, 0, len(union))
			for k := range union {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			merged := map[string]any{}
			for k, v := range h {
				merged[k] = v
			}
			ui := make([]any, len(keys))
			for i, k := range keys {
				ui[i] = k
			}
			merged["used_in"] = ui
			hooksByURL[u] = merged
		} else {
			urlOrder = append(urlOrder, u)
			hooksByURL[u] = h
		}
	}
	mergedHooks := make([]any, 0, len(urlOrder))
	for _, u := range urlOrder {
		mergedHooks = append(mergedHooks, hooksByURL[u])
	}

	// --- evidence-derived band ceilings (code-enforced: the band is NOT the
	// agent's self-report). "high" requires proof-of-life the store can see:
	// at least one dated hook ≤60 days old that is not a website positioning
	// line, and — when a readable Facebook profile is on file — at least one
	// facebook-sourced hook (websites go stale; social is where recency lives).
	strongRecent := 0
	websiteStampedToday := false
	fbRead := false
	cutoffRecent := isoDaysAgoFrom(hookRecencyDays, now)[:10]
	// Seeds the OPERATOR handed us (a reel/post they saw, judged relevant and
	// saved) are pre-qualified by that human act: their AGE is irrelevant, the
	// signal was verified when it was chosen. A hook citing one of those URLs is
	// therefore exempt from the recency rule. Hooks the SYSTEM found on its own
	// are not exempt — they still have to be recent. Provenance comes from the
	// stored seed, not from anything the dossier claims, so the agent cannot
	// grant itself the exemption.
	operatorSeeds := map[string]bool{}
	for _, sd := range mapsOf(mList(mMap(prevContact, "identities"), "seeds")) {
		if mStr(sd, "source") == "import" {
			if u := normalizeSocial(mStr(sd, "url")); u != "" {
				operatorSeeds[u] = true
			}
		}
	}
	for _, hv := range mergedHooks {
		h, ok := hv.(map[string]any)
		if !ok {
			continue
		}
		od := mStr(h, "observed_date")
		htype := mStr(h, "type")
		fromOperatorSeed := operatorSeeds[normalizeSocial(mStr(h, "evidence_url"))]
		if fromOperatorSeed || (len(od) >= 10 && od[:10] >= cutoffRecent && htype != "website_update") {
			strongRecent++
		}
		if htype == "website_update" && len(od) >= 10 && od[:10] == now[:10] {
			websiteStampedToday = true
		}
		eu := strings.ToLower(mStr(h, "evidence_url"))
		if strings.Contains(eu, "facebook.com") || strings.Contains(eu, "fb.watch") || strings.Contains(eu, "fb.com") {
			fbRead = true
		}
	}
	if websiteStampedToday {
		problems = append(problems, "website_update observed_date equals the run date — that is when the page was READ, not when its content changed; use the content's own publish/update date or leave it empty (undated hooks never count as recent)")
	}
	if band == "high" && strongRecent == 0 {
		band = "review_carefully"
		problems = append(problems, "band capped to review_carefully: high requires >=1 dated hook within the recency window that is not website_update, or a hook citing a URL the operator supplied — a website positioning line alone is not proof-of-life")
	}
	// a facebook profile counts whether it was already on file OR surfaced in
	// this same dossier (identity.profiles / channels_found.profiles)
	fbProfile := mStr(mMap(mMap(prevContact, "identities"), "socials"), "facebook")
	if fbProfile == "" {
		fbProfile = mStr(mMap(ident, "profiles"), "facebook")
	}
	if fbProfile == "" {
		fbProfile = mStr(mMap(mMap(ident, "channels_found"), "profiles"), "facebook")
	}
	if band == "high" && fbProfile != "" && !fbRead {
		band = "review_carefully"
		problems = append(problems, "band capped to review_carefully: a Facebook profile is on file but no facebook-sourced hook exists — read it (collector fb.profile.header/posts/videos) before claiming high confidence")
	}
	rankedAngles := mList(brief, "ranked_angles")
	if rankedAngles == nil {
		rankedAngles = []any{}
	}
	enrichment := map[string]any{
		"identity": mergedIdent,
		"context":  orEmptyMap(mMap(dossier, "context")),
		"hooks":    mergedHooks, "hooks_refreshed_at": now,
		"writing_brief": map[string]any{"one_liner": mStr(brief, "one_liner"),
			"ranked_angles": rankedAngles, "do_not_mention": doNotMention,
			"personalization_confidence": conf},
		"confidence_band":       band,
		"email_not_found_at":    prevEn["email_not_found_at"],
		"no_verifiable_hook_at": prevEn["no_verifiable_hook_at"],
	}
	if truthy(dossier["mark_email_not_found"]) {
		// Refuse the CONCLUSION, keep the WORK: a dossier whose every evidence URL
		// is on facebook.com never left the Facebook pass, so it cannot declare the
		// address unfindable. The hooks it did gather are still written; only the
		// negative cache is withheld, so the lead stays queued as email_discovery
		// instead of disappearing into "skipped".
		fbProf := mStr(mMap(mMap(prevContact, "identities"), "socials"), "facebook")
		if fbProf == "" {
			fbProf = mStr(mMap(ident, "profiles"), "facebook")
		}
		if fbProf == "" {
			fbProf = mStr(mMap(mMap(ident, "channels_found"), "profiles"), "facebook")
		}
		switch {
		case fbProf != "" && !dossierReadAboutTab(dossier, ident):
			// The highest-yield surface for an address is the profile's About /
			// Contact-info tab, and the collector only reads the page it is pointed
			// at. A pass that resolved a reel owner and then jumped straight to web
			// search never opened the one page where the address usually sits: two
			// LeadUp profiles published their address there and still came back
			// "no email". If we know the profile, that tab must have been visited.
			problems = append(problems, "mark_email_not_found REFUSED: a Facebook profile is on file but nothing in this dossier shows the profile was dug for an address. Run ONE collector job — capability fb.profile.contacts on the PROFILE url — and copy its record into the dossier as email_discovery {checked, found_on, emails, websites}. That capability walks the bio (expanding \"See more\") and the About sub-tabs contact_info/intro/basic_info/links itself; its `checked` list is the audit trail this gate reads. Only after it comes back empty is the website hop worth spending")
		case dossierLookedBeyondFacebook(dossier, ident):
			enrichment["email_not_found_at"] = now
		default:
			problems = append(problems, "mark_email_not_found REFUSED: every evidence URL in this dossier is on facebook.com, so the email hunt never left the Facebook pass — follow the official website / About-Contact page / licence roster or directory / web search / reverse search before declaring an address unfindable (the hooks you gathered were kept, the lead stays queued for email discovery)")
		}
	}
	// The operator's own curated seed IS hook material: they saved that reel/post
	// as the reason to contact this person. On the last live batch 58 leads
	// carried a resolved reel seed, zero hooks, and NO declaration of any kind —
	// the dossier simply came back with an empty array and the run reported them
	// "skipped". So this checks the SILENT case, not the declared one:
	// `mark_no_hook` was set on exactly 0 of those 58. One of them had already
	// opened the reel to resolve its owner and then discarded the content.
	unharvested := unharvestedSeedHooks(prevContact, mergedHooks)
	if len(mergedHooks) == 0 && len(unharvested) > 0 {
		problems = append(problems, fmt.Sprintf("no hooks recorded, but this lead's own curated content is still unread: %s. The operator SAVED that link as the reason to reach this person — open it, confirm the owner matches and url_drifted is false, and record the professional detail as a hook whose evidence_url IS that url. Hunt a substitute only once the content is unreadable, drifted, or carries nothing professional", strings.Join(unharvested, ", ")))
	}
	if len(usableHooks) == 0 && truthy(dossier["mark_no_hook"]) {
		if len(unharvested) > 0 {
			problems = append(problems, "mark_no_hook REFUSED: the springboard is not exhausted while the operator's own saved content is unread — read the seed first (the hooks you did gather were kept)")
		} else {
			enrichment["no_verifiable_hook_at"] = now
		}
	}
	patch := map[string]any{"enrichment": enrichment}

	// Promote the discovered name to the canonical contact.name, which is what
	// every consumer (UI list, greeting, name_overuse gate) reads. Only when the
	// contact has none: an operator-supplied name always outranks an observed
	// one. Junk is refused with the reason rather than stored — blank displays as
	// the id, which is ugly; a tagline displays as the reader's NAME in a real
	// greeting, which is worse.
	if strings.TrimSpace(mStr(mMap(prevContact, "name"), "full")) == "" {
		if observed := strings.TrimSpace(mStr(ident, "name")); observed != "" {
			if ok, why := displayNameOK(observed); ok {
				np := map[string]any{"full": observed, "source": "enrich"}
				if g := strings.TrimSpace(mStr(ident, "name_given")); g != "" {
					np["given"] = g
				}
				if et := strings.TrimSpace(mStr(ident, "entity_type")); et != "" {
					np["entity_type"] = et
				}
				np["greetable"] = greetableName(observed) || mStr(ident, "name_given") != ""
				patch["name"] = np
			} else {
				problems = append(problems, fmt.Sprintf("identity.name %s rejected: %s. Give the profile's actual name (a person's name, or the page/company name) — the value ends up in the CRM list and in the email greeting, so a tagline or page-chrome string must not be stored as a name", pyRepr(observed), why))
			}
		}
	}
	// A person's name in current_company is the header processor's improvisation
	// showing through: 182 live contacts carry the same string in both. The
	// dossier now has identity.name for the name; current_company means the
	// employer and nothing else.
	if cc := strings.TrimSpace(mStr(ident, "current_company")); cc != "" &&
		strings.EqualFold(cc, strings.TrimSpace(mStr(ident, "name"))) &&
		mStr(ident, "entity_type") == "person" {
		problems = append(problems, "identity.current_company duplicates identity.name for a PERSON — current_company is the employer, not the person. Leave it empty when the employer is unknown")
	}

	found := withEmailDiscovery(mMap(ident, "channels_found"), mMap(dossier, "email_discovery"))
	// A later enrich that SUCCEEDS must CLEAR the matching negative cache. Both
	// flags were carried forward unconditionally, so once a lead was marked
	// no-hook / email-not-found it kept that mark even after a later pass found
	// what it lacked — and would then be skipped for the whole retry window while
	// holding perfectly good data.
	// NOTE: clearing must WRITE an empty value, not delete the key — the patch is
	// merged with deepUpdate, which only overwrites keys present in the patch and
	// leaves absent ones untouched, so a delete here would never reach the record.
	if len(mergedHooks) > 0 {
		enrichment["no_verifiable_hook_at"] = ""
	}
	if len(mList(found, "emails")) > 0 {
		enrichment["email_not_found_at"] = ""
	}
	var emails, phones []any
	for _, e := range mList(found, "emails") {
		if s, ok := e.(string); ok && s != "" && !skipIdent["email|"+normalizeEmail(s)] {
			emails = append(emails, map[string]any{"address": s, "source": "enrich", "status": "unverified"})
		}
	}
	for _, p := range mList(found, "phones") {
		if s, ok := p.(string); ok && s != "" && !skipIdent["phone|"+normalizePhone(s)] {
			phones = append(phones, map[string]any{"number": s, "type": "cell", "source": "enrich"})
		}
	}
	// Found PROFILES become canonical identities too (same rule as found
	// emails/phones — never siloed in the dossier): channels_found.profiles is
	// {platform: url} and/or a plain list of urls (platform-classified here).
	foundSocials := map[string]any{}
	foundWebsite := any(nil)
	switch pv := found["profiles"].(type) {
	case map[string]any:
		for platform, u := range pv {
			if s, ok := u.(string); ok && s != "" {
				if platform == "website" {
					foundWebsite = s
					continue
				}
				// The map form used to trust the caller completely: whatever string
				// sat under "facebook" became the person's Facebook identity. A live
				// run stored SEARCH-RESULT URLs there, and because normalizeSocial
				// drops the query string every Bing search collapsed to the single
				// key "bing.com/search" — so dozens of unrelated leads shared one
				// identity and consolidation chain-merged 54 reels into one contact.
				// A profile identity must now LOOK like a profile and match the
				// platform it is filed under; the classifier decides, not the caller.
				kind, plat, _ := classifyLeadURLFull(s)
				if kind != "profile" || plat == "" || (platform != "" && plat != platform) {
					problems = append(problems, fmt.Sprintf("channels_found.profiles[%s] rejected: %s is not a %s PROFILE url (classified %s/%s) — a search-result, post or directory link is evidence, never an identity; give the profile page itself", pyRepr(platform), pyRepr(s), platform, kind, plat))
					continue
				}
				if !skipIdent["social|"+normalizeSocial(s)] {
					foundSocials[plat] = s
				}
			}
		}
	case []any:
		for _, u := range pv {
			if s, ok := u.(string); ok && s != "" {
				kind, platform := classifyLeadURL(s)
				switch kind {
				case "profile":
					if !skipIdent["social|"+normalizeSocial(s)] {
						foundSocials[platform] = s
					}
				case "website":
					foundWebsite = s
				}
			}
		}
	}
	if len(emails) > 0 || len(phones) > 0 || len(foundSocials) > 0 || foundWebsite != nil {
		idPatch := map[string]any{"emails": orEmptyList(emails), "phones": orEmptyList(phones)}
		if len(foundSocials) > 0 {
			idPatch["socials"] = foundSocials
		}
		if foundWebsite != nil {
			idPatch["website"] = foundWebsite
		}
		patch["identities"] = idPatch
	}
	// A resolved origin closes the seeds: once any profile/email anchor was
	// found for this contact, its content seeds are no longer "unresolved".
	if len(foundSocials) > 0 || foundWebsite != nil || len(emails) > 0 {
		resolvedTo := ""
		for _, k := range sortedKeys(foundSocials) {
			resolvedTo = fmt.Sprint(foundSocials[k])
			break
		}
		if resolvedTo == "" && foundWebsite != nil {
			resolvedTo = fmt.Sprint(foundWebsite)
		}
		prevSeeds := mapsOf(mList(mMap(c.getContact(leadID), "identities"), "seeds"))
		if len(prevSeeds) > 0 {
			var seedPatch []any
			for _, sd := range prevSeeds {
				cp := map[string]any{}
				for k, v := range sd {
					cp[k] = v
				}
				if mStr(cp, "status") != "resolved" {
					cp["status"] = "resolved"
					if resolvedTo != "" {
						cp["resolved_profile"] = resolvedTo
					}
				}
				seedPatch = append(seedPatch, cp)
			}
			ip, ok := patch["identities"].(map[string]any)
			if !ok {
				ip = map[string]any{}
				patch["identities"] = ip
			}
			ip["seeds"] = seedPatch
		}
	}
	if mStr(ident, "still_active") == "confirmed" {
		ch, _ := patch["channels"].(map[string]any)
		if ch == nil {
			ch = map[string]any{}
			patch["channels"] = ch
		}
		if _, ok := ch["email"]; !ok {
			ch["email"] = map[string]any{}
		}
	}
	if _, err := c.setContact(leadID, patch); err != nil {
		return nil, err
	}
	res := map[string]any{"lead_id": leadID, "usable_hooks": len(usableHooks), "confidence_band": band,
		"do_not_mention": len(doNotMention), "problems": problems}
	if len(consolidated) > 0 {
		res["consolidated"] = consolidated
	}
	if len(dupSuspects) > 0 {
		res["duplicate_suspected"] = dupSuspects
	}
	return res, nil
}

func hookType(h map[string]any) string {
	if t := mStr(h, "type"); t != "" {
		return t
	}
	if h["type"] != nil {
		return fmt.Sprint(h["type"])
	}
	return "?"
}

// pyRepr renders a string like Python repr in the messages tests match on.
func pyRepr(s string) string { return "'" + s + "'" }

// floatOrNil mirrors Python float(x)-or-ValueError: numbers pass through,
// numeric strings parse, junk reports false.
func floatOrNil(v any) (float64, bool) {
	if f, ok := toFloat(v); ok {
		return f, true
	}
	if s, ok := v.(string); ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

func orEmptyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func orEmptyList(l []any) []any {
	if l == nil {
		return []any{}
	}
	return l
}

// --- contacts + identity + merge -----------------------------------------------

func (c *crmStore) resolve(leadID string) string {
	seen := map[string]bool{}
	cur := leadID
	for cur != "" && !seen[cur] {
		seen[cur] = true
		rec, err := c.a.get("contacts", cur)
		if err != nil || rec == nil {
			return cur
		}
		merge := mMap(rec, "merge")
		mergedInto := mStr(merge, "merged_into")
		if mStr(merge, "status") == "merged" && mergedInto != "" {
			cur = mergedInto
		} else {
			return cur
		}
	}
	return cur
}

func (c *crmStore) getContact(leadID string) map[string]any {
	rec, err := c.a.get("contacts", leadID)
	if err != nil || rec == nil {
		return nil
	}
	if mStr(mMap(rec, "merge"), "status") == "merged" {
		r2, err := c.a.get("contacts", c.resolve(leadID))
		if err != nil {
			return nil
		}
		return r2
	}
	return rec
}

// identityPairs yields (kind, normalized value) pairs in Python's iteration
// order: emails, phones, then socials (socials map order is Go-random; the
// consumers treat the pairs as a set, and register/suppress are per-pair).
func (c *crmStore) identityPairs(contact map[string]any) [][2]string {
	var out [][2]string
	ids := mMap(contact, "identities")
	for _, e := range mapsOf(mList(ids, "emails")) {
		if v := normalizeEmail(mStr(e, "address")); v != "" {
			out = append(out, [2]string{"email", v})
		}
	}
	for _, p := range mapsOf(mList(ids, "phones")) {
		if v := normalizePhone(mStr(p, "number")); v != "" {
			out = append(out, [2]string{"phone", v})
		}
	}
	socials := mMap(ids, "socials")
	keys := make([]string, 0, len(socials))
	for k := range socials {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic (set-semantics downstream)
	for _, k := range keys {
		if s, ok := socials[k].(string); ok {
			if v := normalizeSocial(s); v != "" {
				out = append(out, [2]string{"social", v})
			}
		}
	}
	// content seeds are dedupe identities too: the same reel/post pasted twice
	// is the same lead, even before anyone knows who owns it
	for _, sd := range mapsOf(mList(ids, "seeds")) {
		if v := normalizeSocial(mStr(sd, "url")); v != "" {
			out = append(out, [2]string{"seed", v})
		}
	}
	return out
}

// hasContactAnchor reports whether the contact has any resolvable identity
// anchor (email / phone / profile / website) as opposed to content seeds only.
func hasContactAnchor(contact map[string]any) bool {
	ids := mMap(contact, "identities")
	for _, e := range mapsOf(mList(ids, "emails")) {
		if validEmail(mStr(e, "address")) {
			return true
		}
	}
	for _, p := range mapsOf(mList(ids, "phones")) {
		if normalizePhone(mStr(p, "number")) != "" {
			return true
		}
	}
	for _, v := range mMap(ids, "socials") {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return true
		}
	}
	if s, ok := ids["website"].(string); ok && strings.TrimSpace(s) != "" {
		return true
	}
	return false
}

func contactSkeleton(leadID string) map[string]any {
	return map[string]any{
		"id": leadID, "schema_version": 2, "created_at": nowISO(), "updated_at": nowISO(),
		"name": map[string]any{"full": "", "first": "", "last": ""}, "account_id": "",
		// socials is an OPEN map (any platform key: facebook, youtube, tiktok,
		// zillow, gbp, ...); seeds hold unique CONTENT clues (reel/video/post/
		// blog URLs) that enrichment must resolve back to their owner profile.
		"identities": map[string]any{"emails": []any{}, "phones": []any{},
			"socials": map[string]any{"facebook": nil, "instagram": nil, "linkedin": nil, "zalo": nil, "x": nil},
			"website": nil, "seeds": []any{}},
		"channels": map[string]any{
			"email":     map[string]any{"status": "needs_data"},
			"sms":       map[string]any{"status": "needs_optin", "mode": "assisted"},
			"messenger": map[string]any{"status": "needs_data", "mode": "assisted"},
			"zalo":      map[string]any{"status": "needs_data", "mode": "assisted"}},
		"lifecycle_stage": "lead", "tz": "", "tags": []any{}, "custom_fields": map[string]any{},
		"owner": "agency", "enrichment": map[string]any{}, "assigned_sendbox": nil,
		"sequence_state": "active",
		"merge":          map[string]any{"status": "active", "merged_into": nil},
		"next_action":    map[string]any{"task_id": nil},
	}
}

func mergeIntoContact(rec, patch map[string]any) {
	for k, v := range patch {
		if k == "id" {
			continue
		}
		if k == "identities" {
			vm, ok := v.(map[string]any)
			if !ok {
				rec[k] = v
				continue
			}
			ids := mMap(rec, "identities")
			if ids == nil {
				ids = map[string]any{}
				rec["identities"] = ids
			}
			emails := mList(ids, "emails")
			for _, e := range mapsOf(mList(vm, "emails")) {
				norm := normalizeEmail(mStr(e, "address"))
				if norm == "" {
					continue
				}
				dup := false
				for _, x := range mapsOf(emails) {
					if normalizeEmail(mStr(x, "address")) == norm {
						dup = true
						break
					}
				}
				if !dup {
					cp := map[string]any{}
					for ek, ev := range e {
						cp[ek] = ev
					}
					cp["address"] = norm
					emails = append(emails, cp)
				}
			}
			ids["emails"] = orEmptyList(emails)
			phones := mList(ids, "phones")
			for _, p := range mapsOf(mList(vm, "phones")) {
				norm := normalizePhone(mStr(p, "number"))
				if norm == "" {
					continue
				}
				dup := false
				for _, x := range mapsOf(phones) {
					if normalizePhone(mStr(x, "number")) == norm {
						dup = true
						break
					}
				}
				if !dup {
					cp := map[string]any{}
					for pk, pv := range p {
						cp[pk] = pv
					}
					cp["number"] = norm
					phones = append(phones, cp)
				}
			}
			ids["phones"] = orEmptyList(phones)
			for sk, sv := range mMap(vm, "socials") {
				if truthy(sv) {
					soc := mMap(ids, "socials")
					if soc == nil {
						soc = map[string]any{}
						ids["socials"] = soc
					}
					soc[sk] = sv
				}
			}
			if truthy(vm["website"]) {
				ids["website"] = vm["website"]
			}
			seeds := mList(ids, "seeds")
			for _, sd := range mapsOf(mList(vm, "seeds")) {
				norm := normalizeSocial(mStr(sd, "url"))
				if norm == "" {
					continue
				}
				merged := false
				for _, x := range mapsOf(seeds) {
					if normalizeSocial(mStr(x, "url")) == norm {
						for sk, sv := range sd { // resubmit wins (e.g. status -> resolved)
							x[sk] = sv
						}
						merged = true
						break
					}
				}
				if !merged {
					cp := map[string]any{}
					for sk, sv := range sd {
						cp[sk] = sv
					}
					// Provenance is stamped by the STORE, not taken from the caller:
					// a seed arriving on an add/import patch came from the operator
					// (they pasted or imported that URL). A future discovery path that
					// mints seeds on its own must stamp its own source and is NOT
					// covered by the operator-seed recency exemption.
					if mStr(cp, "source") == "" {
						cp["source"] = "import"
					}
					if mStr(cp, "status") == "" {
						cp["status"] = "unresolved"
					}
					seeds = append(seeds, cp)
				}
			}
			if seeds != nil {
				ids["seeds"] = seeds
			}
			continue
		}
		if vm, ok := v.(map[string]any); ok {
			if rm, ok := rec[k].(map[string]any); ok {
				deepUpdate(rm, vm)
				continue
			}
		}
		rec[k] = v
	}
}

func deepUpdate(dst, src map[string]any) {
	for k, v := range src {
		if vm, ok := v.(map[string]any); ok {
			if dm, ok := dst[k].(map[string]any); ok {
				deepUpdate(dm, vm)
				continue
			}
		}
		dst[k] = v
	}
}

func (c *crmStore) addContact(fields map[string]any) (string, string, error) {
	unlock, err := c.a.lock("contacts_add")
	if err != nil {
		return "", "", err
	}
	defer unlock()
	for _, pair := range c.identityPairs(fields) {
		if existing := c.a.findByIdentity(pair[0], pair[1]); existing != "" {
			matchedID := c.resolve(existing)
			// A matched row still carries information (another reel/seed, a
			// second phone, a tag). Union it in — dedupe must never lose
			// loaded data — instead of dropping the row on the floor.
			if err := c.unionMatchedRow(matchedID, fields); err != nil {
				return "", "", err
			}
			return matchedID, "matched", nil
		}
	}
	leadID := mStr(fields, "id")
	if leadID == "" {
		leadID = newULID("c_")
	}
	rec := contactSkeleton(leadID)
	mergeIntoContact(rec, fields)
	if err := c.a.put("contacts", leadID, rec); err != nil {
		return "", "", err
	}
	for _, pair := range c.identityPairs(rec) {
		if err := c.a.registerIdentity(pair[0], pair[1], leadID); err != nil {
			return "", "", err
		}
	}
	return leadID, "created", nil
}

// unionMatchedRow folds a re-imported row into the contact it matched.
// Whitelist-union only: identities (emails/phones/seeds append-dedup;
// socials/website fill-if-empty so import junk never clobbers an enriched
// canonical URL), name fill-if-empty, tags union, custom_fields
// fill-if-missing. Import rows never touch workflow state.
func (c *crmStore) unionMatchedRow(leadID string, fields map[string]any) error {
	rec, err := c.a.get("contacts", leadID)
	if err != nil || rec == nil {
		return nil
	}
	patch := map[string]any{}
	if fi, ok := fields["identities"].(map[string]any); ok {
		ip := map[string]any{}
		for _, k := range []string{"emails", "phones", "seeds"} {
			if v, ok := fi[k]; ok {
				ip[k] = v
			}
		}
		recIDs := mMap(rec, "identities")
		recSoc := mMap(recIDs, "socials")
		soc := map[string]any{}
		for k, v := range mMap(fi, "socials") {
			if truthy(v) && !truthy(recSoc[k]) {
				soc[k] = v
			}
		}
		if len(soc) > 0 {
			ip["socials"] = soc
		}
		if truthy(fi["website"]) && !truthy(recIDs["website"]) {
			ip["website"] = fi["website"]
		}
		if len(ip) > 0 {
			patch["identities"] = ip
		}
	}
	if mStr(mMap(rec, "name"), "full") == "" {
		if fn, ok := fields["name"].(map[string]any); ok && mStr(fn, "full") != "" {
			patch["name"] = fn
		}
	}
	if ft := mList(fields, "tags"); len(ft) > 0 {
		tags := mList(rec, "tags")
		have := map[string]bool{}
		for _, t := range tags {
			have[fmt.Sprint(t)] = true
		}
		for _, t := range ft {
			if !have[fmt.Sprint(t)] {
				tags = append(tags, t)
				have[fmt.Sprint(t)] = true
			}
		}
		patch["tags"] = orEmptyList(tags)
	}
	if fcf := mMap(fields, "custom_fields"); len(fcf) > 0 {
		rcf := mMap(rec, "custom_fields")
		add := map[string]any{}
		for k, v := range fcf {
			if _, ok := rcf[k]; !ok {
				add[k] = v
			}
		}
		if len(add) > 0 {
			patch["custom_fields"] = add
		}
	}
	if len(patch) == 0 {
		return nil
	}
	_, err = c.setContact(leadID, patch)
	return err
}

func (c *crmStore) setContact(leadID string, patch map[string]any) (map[string]any, error) {
	leadID = c.resolve(leadID)
	rec, err := c.a.update("contacts", leadID, func(rec map[string]any) map[string]any {
		mergeIntoContact(rec, patch)
		// `channels.email.status` is seeded "needs_data" when a contact is created
		// and nothing ever cleared it, so it kept reading "needs_data" long after a
		// real address was on file. A live run read that as a blocker and refused to
		// draft 28 contacts that were perfectly draftable. Keep the field truthful:
		// once a valid address exists the channel is usable. Suppression states
		// (opted_out / bounced) are terminal and never overwritten here.
		if contactHasUsableEmail(rec) {
			chans := mMap(rec, "channels")
			if chans == nil {
				chans = map[string]any{}
				rec["channels"] = chans
			}
			em, _ := chans["email"].(map[string]any)
			if em == nil {
				em = map[string]any{}
				chans["email"] = em
			}
			if s := mStr(em, "status"); s == "" || s == "needs_data" {
				em["status"] = "usable"
			}
		}
		return rec
	})
	if err != nil {
		return nil, err
	}
	for _, pair := range c.identityPairs(rec) {
		if c.a.findByIdentity(pair[0], pair[1]) != leadID {
			if err := c.a.registerIdentity(pair[0], pair[1], leadID); err != nil {
				return nil, err
			}
		}
	}
	return rec, nil
}

func (c *crmStore) merge(loserID, winnerID string) (map[string]any, error) {
	loserID, winnerID = c.resolve(loserID), c.resolve(winnerID)
	if loserID == winnerID {
		rec, _ := c.a.get("contacts", winnerID)
		return rec, nil
	}
	loser, err := c.a.get("contacts", loserID)
	if err != nil || loser == nil {
		return nil, storageErrf("loser %s not found", loserID)
	}
	win, err := c.a.update("contacts", winnerID, func(win map[string]any) map[string]any {
		li := mMap(win, "identities")
		if li == nil {
			li = map[string]any{}
			win["identities"] = li
		}
		lo := mMap(loser, "identities")
		emails := mList(li, "emails")
		haveE := map[string]bool{}
		for _, x := range mapsOf(emails) {
			haveE[normalizeEmail(mStr(x, "address"))] = true
		}
		for _, e := range mapsOf(mList(lo, "emails")) {
			n := normalizeEmail(mStr(e, "address"))
			if !haveE[n] {
				emails = append(emails, e)
				haveE[n] = true
			}
		}
		li["emails"] = orEmptyList(emails)
		phones := mList(li, "phones")
		haveP := map[string]bool{}
		for _, x := range mapsOf(phones) {
			haveP[normalizePhone(mStr(x, "number"))] = true
		}
		for _, p := range mapsOf(mList(lo, "phones")) {
			n := normalizePhone(mStr(p, "number"))
			if !haveP[n] {
				phones = append(phones, p)
				haveP[n] = true
			}
		}
		li["phones"] = orEmptyList(phones)
		for k, v := range mMap(lo, "socials") {
			if truthy(v) {
				soc := mMap(li, "socials")
				if soc == nil {
					soc = map[string]any{}
					li["socials"] = soc
				}
				if !truthy(soc[k]) {
					soc[k] = v
				}
			}
		}
		for ch, cval := range mMap(loser, "channels") {
			cm, ok := cval.(map[string]any)
			if !ok {
				continue
			}
			st := mStr(cm, "status")
			if st == "opted_out" || st == "bounced" {
				chans := mMap(win, "channels")
				if chans == nil {
					chans = map[string]any{}
					win["channels"] = chans
				}
				dst, ok := chans[ch].(map[string]any)
				if !ok {
					dst = map[string]any{}
					chans[ch] = dst
				}
				dst["status"] = st
			}
		}
		// FULL UNION — consolidation must never lose loaded data (many reels
		// of one person fold into one record with everything both sides had).
		// seeds: dedup by URL; on dup fill missing keys, resolved state wins
		seeds := mList(li, "seeds")
		for _, sd := range mapsOf(mList(lo, "seeds")) {
			norm := normalizeSocial(mStr(sd, "url"))
			if norm == "" {
				continue
			}
			var hit map[string]any
			for _, x := range mapsOf(seeds) {
				if normalizeSocial(mStr(x, "url")) == norm {
					hit = x
					break
				}
			}
			if hit == nil {
				cp := map[string]any{}
				for k, v := range sd {
					cp[k] = v
				}
				seeds = append(seeds, cp)
				continue
			}
			for k, v := range sd {
				if _, ok := hit[k]; !ok {
					hit[k] = v
				}
			}
			if mStr(sd, "status") == "resolved" && mStr(hit, "status") != "resolved" {
				hit["status"] = "resolved"
				if rp := mStr(sd, "resolved_profile"); rp != "" {
					hit["resolved_profile"] = rp
				}
			}
		}
		if seeds != nil {
			li["seeds"] = seeds
		}
		if !truthy(li["website"]) && truthy(lo["website"]) {
			li["website"] = lo["website"]
		}
		if mStr(mMap(win, "name"), "full") == "" {
			if ln := mMap(loser, "name"); mStr(ln, "full") != "" {
				win["name"] = ln
			}
		}
		tags := mList(win, "tags")
		haveT := map[string]bool{}
		for _, t := range tags {
			haveT[fmt.Sprint(t)] = true
		}
		for _, t := range mList(loser, "tags") {
			if !haveT[fmt.Sprint(t)] {
				tags = append(tags, t)
				haveT[fmt.Sprint(t)] = true
			}
		}
		if tags != nil {
			win["tags"] = tags
		}
		if lcf := mMap(loser, "custom_fields"); len(lcf) > 0 {
			wcf := mMap(win, "custom_fields")
			if wcf == nil {
				wcf = map[string]any{}
				win["custom_fields"] = wcf
			}
			for k, v := range lcf {
				if _, ok := wcf[k]; !ok {
					wcf[k] = v
				}
			}
		}
		// enrichment: hooks union by evidence_url (used_in unioned, winner's
		// content wins on dup); other keys fill-if-missing; freshest stamp
		if loEn := mMap(loser, "enrichment"); len(loEn) > 0 {
			wEn := mMap(win, "enrichment")
			if wEn == nil {
				wEn = map[string]any{}
				win["enrichment"] = wEn
			}
			hooks := mList(wEn, "hooks")
			byURL := map[string]map[string]any{}
			for _, h := range mapsOf(hooks) {
				byURL[mStr(h, "evidence_url")] = h
			}
			for _, h := range mapsOf(mList(loEn, "hooks")) {
				u := mStr(h, "evidence_url")
				if ex, ok := byURL[u]; ok {
					used := mList(ex, "used_in")
					haveU := map[string]bool{}
					for _, x := range used {
						haveU[fmt.Sprint(x)] = true
					}
					for _, x := range mList(h, "used_in") {
						if !haveU[fmt.Sprint(x)] {
							used = append(used, x)
						}
					}
					ex["used_in"] = orEmptyList(used)
					continue
				}
				hooks = append(hooks, h)
				byURL[u] = h
			}
			if hooks != nil {
				wEn["hooks"] = hooks
			}
			for _, k := range []string{"identity", "context", "writing_brief", "confidence_band",
				"email_not_found_at", "no_verifiable_hook_at"} {
				if _, ok := wEn[k]; !ok {
					if v, ok2 := loEn[k]; ok2 {
						wEn[k] = v
					}
				}
			}
			if l := mStr(loEn, "hooks_refreshed_at"); l > mStr(wEn, "hooks_refreshed_at") {
				wEn["hooks_refreshed_at"] = l
			}
		}
		if mStr(loser, "sequence_state") == "frozen" {
			win["sequence_state"] = "frozen"
		}
		if !truthy(win["assigned_sendbox"]) && truthy(loser["assigned_sendbox"]) {
			win["assigned_sendbox"] = loser["assigned_sendbox"]
		}
		// merging the pair settles any duplicate suspicion between them
		if ds := mList(win, "duplicate_suspects"); len(ds) > 0 {
			var keep []any
			for _, d := range ds {
				if dm, ok := d.(map[string]any); ok {
					if oid := mStr(dm, "id"); oid == loserID || oid == winnerID {
						continue
					}
				}
				keep = append(keep, d)
			}
			win["duplicate_suspects"] = orEmptyList(keep)
		}
		return win
	})
	if err != nil {
		return nil, err
	}
	if _, err := c.a.update("contacts", loserID, func(r map[string]any) map[string]any {
		r["merge"] = map[string]any{"status": "merged", "merged_into": winnerID}
		return r
	}); err != nil {
		return nil, err
	}
	for _, pair := range c.identityPairs(loser) {
		if err := c.a.registerIdentity(pair[0], pair[1], winnerID); err != nil {
			return nil, err
		}
	}
	if _, err := c.logActivity("merged", winnerID, fmt.Sprintf("merged %s into %s", loserID, winnerID),
		"agent", nil, map[string]any{"path": loserID}); err != nil {
		return nil, err
	}
	return win, nil
}

// maxConsolidationChain caps how many records one identity may absorb
// automatically. Beyond it the identity is treated as suspect rather than
// authoritative: a genuine "many reels, one creator" cluster is plausible at this
// size, an identity swallowing more is far more likely to be a bad join key.
const maxConsolidationChain = 12

// mergedIntoCount counts tombstones already pointing at this survivor.
func mergedIntoCount(c *crmStore, survivorID string) int {
	rows, err := c.a.query("contacts", []cond{{"merge.status", "=", "merged"}}, nil, -1, 0)
	if err != nil {
		return 0
	}
	n := 0
	for _, doc := range rows {
		if c.resolve(mStr(mMap(doc, "merge"), "merged_into")) == survivorID {
			n++
		}
	}
	return n
}

// anchorScore ranks which record should SURVIVE an auto-consolidation:
// verified reachability first (emails), then phones, profiles, enrichment, name.
func anchorScore(contact map[string]any) int {
	ids := mMap(contact, "identities")
	score := 0
	for _, e := range mapsOf(mList(ids, "emails")) {
		if validEmail(mStr(e, "address")) {
			score += 4
		}
	}
	for _, p := range mapsOf(mList(ids, "phones")) {
		if normalizePhone(mStr(p, "number")) != "" {
			score += 2
		}
	}
	for _, v := range mMap(ids, "socials") {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			score++
		}
	}
	if truthy(ids["website"]) {
		score++
	}
	if len(mMap(contact, "enrichment")) > 0 {
		score += 2
	}
	if mStr(mMap(contact, "name"), "full") != "" {
		score++
	}
	return score
}

// contactsConflict reports when two records look like two DIFFERENT real
// people — both carry a name and the names differ, or both carry emails and
// the sets are disjoint. A shared URL between such records (e.g. a brokerage
// page posting reels of several agents) must FLAG, never auto-merge.
func contactsConflict(a, b map[string]any) bool {
	an := strings.ToLower(strings.TrimSpace(mStr(mMap(a, "name"), "full")))
	bn := strings.ToLower(strings.TrimSpace(mStr(mMap(b, "name"), "full")))
	if an != "" && bn != "" && an != bn {
		return true
	}
	ae := map[string]bool{}
	for _, e := range mapsOf(mList(mMap(a, "identities"), "emails")) {
		if n := normalizeEmail(mStr(e, "address")); n != "" {
			ae[n] = true
		}
	}
	overlap, bHas := false, false
	for _, e := range mapsOf(mList(mMap(b, "identities"), "emails")) {
		if n := normalizeEmail(mStr(e, "address")); n != "" {
			bHas = true
			if ae[n] {
				overlap = true
			}
		}
	}
	return len(ae) > 0 && bHas && !overlap
}

// noteDuplicateSuspects flags BOTH records (append-dedup by other id) and logs
// one activity. Suspected duplicates are excluded from campaign queues until
// the operator merges or clears them.
func (c *crmStore) noteDuplicateSuspects(aID, bID, via, value string) error {
	add := func(onID, otherID string) error {
		_, err := c.a.update("contacts", onID, func(r map[string]any) map[string]any {
			for _, d := range mapsOf(mList(r, "duplicate_suspects")) {
				if mStr(d, "id") == otherID {
					return r
				}
			}
			r["duplicate_suspects"] = append(mList(r, "duplicate_suspects"),
				map[string]any{"id": otherID, "via": via, "value": value, "at": nowISO()})
			return r
		})
		return err
	}
	if err := add(aID, bID); err != nil {
		return err
	}
	if err := add(bID, aID); err != nil {
		return err
	}
	_, err := c.logActivity("duplicate_suspected", aID,
		fmt.Sprintf("possible duplicate of %s (shared %s %s) — records conflict, NOT auto-merged; resolve with `contact merge` (same person) or `contact unsuspect` (different people)", bID, via, value),
		"agent", nil, nil)
	return err
}

// clearDuplicateSuspect removes the mutual flag (operator confirmed the two
// records are different people).
func (c *crmStore) clearDuplicateSuspect(aID, bID string) (map[string]any, error) {
	aID, bID = c.resolve(aID), c.resolve(bID)
	rm := func(onID, otherID string) error {
		_, err := c.a.update("contacts", onID, func(r map[string]any) map[string]any {
			var keep []any
			for _, d := range mList(r, "duplicate_suspects") {
				if dm, ok := d.(map[string]any); ok && mStr(dm, "id") == otherID {
					continue
				}
				keep = append(keep, d)
			}
			r["duplicate_suspects"] = orEmptyList(keep)
			return r
		})
		return err
	}
	if err := rm(aID, bID); err != nil {
		return nil, err
	}
	if err := rm(bID, aID); err != nil {
		return nil, err
	}
	if _, err := c.logActivity("duplicate_cleared", aID,
		fmt.Sprintf("cleared duplicate suspicion with %s (confirmed different people)", bID),
		"human", nil, nil); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "a": aID, "b": bID}, nil
}

// consolidateOnDiscovery: the dossier's channels_found just tied this contact
// to identities that may already belong to OTHER contacts (the "many reels →
// one profile" case). Safe pairs merge immediately — full union, nothing
// loaded is lost — and the dossier lands on the survivor. Conflicting pairs
// are flagged on both records and the conflicting value is withheld from the
// write, so the identity index pointer is never silently stolen. Returns the
// surviving lead id, merge/suspect events for the tool result, and the
// "kind|value" identity keys to withhold.
func (c *crmStore) consolidateOnDiscovery(leadID string, found map[string]any) (string, []any, []any, map[string]bool, error) {
	skip := map[string]bool{}
	var consolidated, suspects []any
	if len(found) == 0 {
		return leadID, nil, nil, skip, nil
	}
	type cand struct{ kind, val string }
	var cands []cand
	for _, e := range mList(found, "emails") {
		if s, ok := e.(string); ok {
			if n := normalizeEmail(s); n != "" {
				cands = append(cands, cand{"email", n})
			}
		}
	}
	for _, p := range mList(found, "phones") {
		if s, ok := p.(string); ok {
			if n := normalizePhone(s); n != "" {
				cands = append(cands, cand{"phone", n})
			}
		}
	}
	switch pv := found["profiles"].(type) {
	case map[string]any:
		for platform, u := range pv {
			if s, ok := u.(string); ok && s != "" && platform != "website" {
				if n := normalizeSocial(s); n != "" {
					cands = append(cands, cand{"social", n})
				}
			}
		}
	case []any:
		for _, u := range pv {
			if s, ok := u.(string); ok && s != "" {
				if kind, _ := classifyLeadURL(s); kind == "profile" {
					if n := normalizeSocial(s); n != "" {
						cands = append(cands, cand{"social", n})
					}
				}
			}
		}
	}
	flagged := map[string]bool{}
	for _, cd := range cands {
		owner := c.a.findByIdentity(cd.kind, cd.val)
		if owner == "" {
			continue
		}
		other := c.resolve(owner)
		cur := c.resolve(leadID)
		if other == cur {
			continue
		}
		curDoc, othDoc := c.getContact(cur), c.getContact(other)
		if curDoc == nil || othDoc == nil {
			continue
		}
		if contactsConflict(curDoc, othDoc) {
			skip[cd.kind+"|"+cd.val] = true
			if !flagged[other] {
				if err := c.noteDuplicateSuspects(cur, other, cd.kind, cd.val); err != nil {
					return cur, consolidated, suspects, skip, err
				}
				suspects = append(suspects, map[string]any{"other_id": other, "via": cd.kind, "value": cd.val})
				flagged[other] = true
			}
			continue
		}
		// Blast-radius guard: consolidation is meant for "several reels of ONE
		// person". If a single identity has already absorbed a crowd, the identity
		// itself is suspect (that is exactly how a shared search-result URL merged
		// 54 unrelated reels into one contact). Stop merging on it and flag both
		// records for the operator instead of growing the pile.
		if len(mapsOf(mList(othDoc, "duplicate_suspects"))) == 0 && mergedIntoCount(c, other) >= maxConsolidationChain {
			skip[cd.kind+"|"+cd.val] = true
			if err := c.noteDuplicateSuspects(cur, other, cd.kind, cd.val); err != nil {
				return cur, consolidated, suspects, skip, err
			}
			suspects = append(suspects, map[string]any{"other_id": other, "via": cd.kind, "value": cd.val,
				"reason": "consolidation_chain_too_long"})
			continue
		}
		winner, loser := cur, other
		cs, os2 := anchorScore(curDoc), anchorScore(othDoc)
		if os2 > cs || (os2 == cs && (mStr(othDoc, "created_at") < mStr(curDoc, "created_at") ||
			(mStr(othDoc, "created_at") == mStr(curDoc, "created_at") && other < cur))) {
			winner, loser = other, cur
		}
		if _, err := c.merge(loser, winner); err != nil {
			return cur, consolidated, suspects, skip, err
		}
		leadID = winner
		consolidated = append(consolidated, map[string]any{"survivor": winner, "merged": loser,
			"via": cd.kind, "value": cd.val})
	}
	return c.resolve(leadID), consolidated, suspects, skip, nil
}

// --- activities / tasks / deals -------------------------------------------------

func (c *crmStore) logActivity(atype, contactID, summary, by string, dealID any, ref map[string]any) (map[string]any, error) {
	var cid any
	if contactID != "" {
		cid = c.resolve(contactID)
	}
	if ref == nil {
		ref = map[string]any{}
	}
	return c.a.appendLog("activities", map[string]any{
		"id": newULID("act_"), "contact_id": cid, "deal_id": dealID,
		"type": atype, "summary": summary, "ref": ref, "by": by,
	})
}

func (c *crmStore) addTask(title string, contactID any, dealID any, dueAt, createdBy, guardKey string) (map[string]any, error) {
	var cid any
	if s, ok := contactID.(string); ok && s != "" {
		cid = c.resolve(s)
	}
	return c.a.appendLog("tasks", map[string]any{
		"id": newULID(""), "contact_id": cid, "deal_id": dealID, "title": title,
		"due_at": dueAt, "status": "open", "created_by": createdBy, "guard_key": guardKey,
	})
}

func (c *crmStore) latestTasks() []map[string]any {
	latest := map[string]map[string]any{}
	var order []string
	rows, _ := c.a.readLog("tasks", -1, nil)
	for _, t := range rows {
		id := mStr(t, "id")
		if _, seen := latest[id]; !seen {
			order = append(order, id)
		}
		latest[id] = t
	}
	out := make([]map[string]any, 0, len(order))
	for _, id := range order {
		out = append(out, latest[id])
	}
	return out
}

func (c *crmStore) openTasksFor(contactID string) []map[string]any {
	cid := c.resolve(contactID)
	var out []map[string]any
	for _, t := range c.latestTasks() {
		if mStr(t, "contact_id") == cid && mStr(t, "status") == "open" {
			out = append(out, t)
		}
	}
	return out
}

func (c *crmStore) hasOpenTask(contactID, title string) bool {
	rc := c.resolve(contactID)
	for _, t := range c.latestTasks() {
		if mStr(t, "status") == "open" && mStr(t, "title") == title && mStr(t, "contact_id") == rc {
			return true
		}
	}
	return false
}

func (c *crmStore) closeTasks(contactID string) (int, error) {
	n := 0
	for _, t := range c.openTasksFor(contactID) {
		cp := map[string]any{}
		for k, v := range t {
			cp[k] = v
		}
		cp["status"] = "cancelled"
		if _, err := c.a.appendLog("tasks", cp); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (c *crmStore) stageProb(pipeline, stage string) float64 {
	for _, p := range mapsOf(mList(c.getPipelines(), "pipelines")) {
		if mStr(p, "id") == pipeline {
			for _, s := range mapsOf(mList(p, "stages")) {
				if mStr(s, "id") == stage {
					return asFloat(s["probability"], 0.0)
				}
			}
		}
	}
	return 0.0
}

func (c *crmStore) createDeal(contactID, stage, pipeline, by, evidenceActivityID string, extra map[string]any) (map[string]any, error) {
	cid := c.resolve(contactID)
	did := newULID("d_")
	if extra == nil {
		extra = map[string]any{}
	}
	value := extra["value"]
	if value == nil {
		value = 0
	}
	currency := mStr(extra, "currency")
	if currency == "" {
		currency = "USD"
	}
	rec := map[string]any{
		"id": did, "schema_version": 1, "name": mStr(extra, "name"),
		"contact_ids": []any{cid}, "account_id": mStr(extra, "account_id"),
		"pipeline": pipeline, "stage": stage, "value": value, "currency": currency,
		"probability":    c.stageProb(pipeline, stage),
		"expected_close": "", "source_campaign": mStr(extra, "source_campaign"),
		"stage_history": []any{map[string]any{"stage": stage, "at": nowISO(), "by": by,
			"evidence_activity_id": evidenceActivityID}},
		"status": "open", "lost_reason": nil, "next_action": map[string]any{"task_id": nil},
	}
	if err := c.a.put("deals", did, rec); err != nil {
		return nil, err
	}
	if _, err := c.logActivity("stage_change", cid, fmt.Sprintf("deal %s created at %s", did, stage),
		by, did, map[string]any{"path": evidenceActivityID}); err != nil {
		return nil, err
	}
	return rec, nil
}

func (c *crmStore) openDealFor(contactID string) map[string]any {
	cid := c.resolve(contactID)
	rows, _ := c.a.query("deals", []cond{{Field: "status", Op: "=", Value: "open"}}, nil, -1, 0)
	for _, d := range rows {
		for _, x := range mList(d, "contact_ids") {
			if s, ok := x.(string); ok && s == cid {
				return d
			}
		}
	}
	return nil
}

func (c *crmStore) moveDeal(dealID, stage, evidenceActivityID, by string) (map[string]any, error) {
	return c.a.update("deals", dealID, func(d map[string]any) map[string]any {
		d["stage"] = stage
		pipeline := mStr(d, "pipeline")
		if pipeline == "" {
			pipeline = "default_sales"
		}
		d["probability"] = c.stageProb(pipeline, stage)
		hist := mList(d, "stage_history")
		hist = append(hist, map[string]any{"stage": stage, "at": nowISO(), "by": by,
			"evidence_activity_id": evidenceActivityID})
		d["stage_history"] = hist
		if stage == "won" {
			d["status"] = "won"
		} else if stage == "lost" {
			d["status"] = "lost"
		}
		return d
	})
}

// --- suppression ----------------------------------------------------------------

func (c *crmStore) globalSuppressionPath() string {
	d := c.clientDir
	for i := 0; i < 6; i++ {
		cand := filepath.Join(d, "suppression", "global_suppression.jsonl")
		if st, err := os.Stat(filepath.Join(d, "clients")); err == nil && st.IsDir() {
			return cand
		}
		if st, err := os.Stat(filepath.Join(d, "storage_config.json")); err == nil && !st.IsDir() {
			return cand
		}
		d = filepath.Dir(d)
	}
	return filepath.Join(c.clientDir, "..", "..", "..", "suppression", "global_suppression.jsonl")
}

func normalizeByKind(kind, value string) string {
	switch kind {
	case "email":
		return normalizeEmail(value)
	case "phone":
		return normalizePhone(value)
	case "social":
		return normalizeSocial(value)
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func (c *crmStore) suppressAdd(kind, value, reason, tier string, scope any, sourceActivityID, by string, tags []any) (map[string]any, error) {
	norm := normalizeByKind(kind, value)
	if scope == nil {
		if tier == "global" {
			scope = "all_clients"
		} else {
			scope = filepath.Base(filepath.Dir(c.clientDir))
		}
	}
	if tags == nil {
		tags = []any{}
	}
	rec := map[string]any{"tier": tier, "match": map[string]any{"kind": kind, "value": norm},
		"reason": reason, "scope": scope, "source_activity_id": sourceActivityID,
		"added_by": by, "tags": tags}
	if tier == "global" {
		gp := c.globalSuppressionPath()
		if err := os.MkdirAll(filepath.Dir(gp), 0o755); err != nil {
			return nil, err
		}
		rec["seq"] = appendJSONLSeq(gp)
		rec["ts"] = nowISO()
		f, err := os.OpenFile(gp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if _, err := f.WriteString(marshalLineJSON(rec) + "\n"); err != nil {
			return nil, err
		}
		return rec, nil
	}
	return c.a.appendLog("suppression", rec)
}

func appendJSONLSeq(path string) int {
	n := 0
	data, err := os.ReadFile(path)
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) != "" {
				n++
			}
		}
	}
	return n + 1
}

func (c *crmStore) suppressContact(contactID, reason, sourceActivityID, by string) (int, error) {
	ct := c.getContact(contactID)
	if ct == nil {
		return 0, nil
	}
	n := 0
	for _, pair := range c.identityPairs(ct) {
		if _, err := c.suppressAdd(pair[0], pair[1], reason, "client", nil, sourceActivityID, by, nil); err != nil {
			return n, err
		}
		n++
	}
	if reason == "unsubscribe" || reason == "reply_negative" || reason == "remove_intent" {
		if _, err := c.setContact(contactID, map[string]any{
			"channels": map[string]any{"email": map[string]any{"status": "opted_out"}}}); err != nil {
			return n, err
		}
	}
	return n, nil
}

func (c *crmStore) isSuppressed(email, phone string, socials []string) map[string]any {
	var wanted [][2]string
	if email != "" {
		wanted = append(wanted, [2]string{"email", normalizeEmail(email)})
		if strings.Contains(email, "@") {
			parts := strings.Split(normalizeEmail(email), "@")
			if dom := parts[len(parts)-1]; dom != "" {
				wanted = append(wanted, [2]string{"domain", dom})
			}
		}
	}
	if phone != "" {
		wanted = append(wanted, [2]string{"phone", normalizePhone(phone)})
	}
	for _, u := range socials {
		wanted = append(wanted, [2]string{"social", normalizeSocial(u)})
	}
	rows, _ := c.a.readLog("suppression", -1, nil)
	rows = append(rows, readJSONLines(c.globalSuppressionPath())...)
	for _, r := range rows {
		m := mMap(r, "match")
		key := [2]string{mStr(m, "kind"), mStr(m, "value")}
		for _, w := range wanted {
			if key == w {
				return r
			}
		}
	}
	return nil
}

// --- deterministic rules engine --------------------------------------------------

func (c *crmStore) guardSeen(ruleID, activityID string) bool {
	key := ruleID + ":" + activityID
	rows, _ := c.a.readLog("_rule_guards", -1, nil)
	for _, g := range rows {
		if mStr(g, "key") == key {
			return true
		}
	}
	return false
}

func (c *crmStore) guardMark(ruleID, activityID string) error {
	_, err := c.a.appendLog("_rule_guards", map[string]any{"key": ruleID + ":" + activityID})
	return err
}

func (c *crmStore) applyRules(events []map[string]any) (map[string]any, error) {
	pipelines := c.getPipelines()
	rules := mList(pipelines, "rules")
	if rules == nil {
		rules = mList(defaultPipelines(), "rules")
	}
	applied := []any{}
	pending := []any{}
	for _, ev := range events {
		etype := mStr(ev, "type")
		cid := mStr(ev, "contact_id")
		aid := mStr(ev, "activity_id")
		if aid == "" {
			aid = fmt.Sprintf("noact:%s:%s", etype, cid)
		}
		for _, rv := range mapsOf(rules) {
			triggers := map[string]bool{}
			for _, t := range strings.Split(mStr(rv, "on"), "|") {
				triggers[t] = true
			}
			if !triggers[etype] {
				continue
			}
			ruleID := mStr(rv, "id")
			if aid != "" && c.guardSeen(ruleID, aid) {
				continue
			}
			for _, av := range mList(rv, "do") {
				action, _ := av.(string)
				res, err := c.doAction(action, ruleID, ev, cid, aid, &pending)
				if err != nil {
					return nil, err
				}
				if res != nil {
					applied = append(applied, map[string]any{"rule": ruleID, "action": action, "result": res})
				}
			}
			if aid != "" {
				if err := c.guardMark(ruleID, aid); err != nil {
					return nil, err
				}
			}
		}
	}
	return map[string]any{"applied": applied, "pending": pending}, nil
}

func parseAction(action string) (string, map[string]string) {
	action = strings.TrimSpace(action)
	i := strings.Index(action, "(")
	if i < 0 {
		return action, map[string]string{}
	}
	name := action[:i]
	j := strings.LastIndex(action, ")")
	inner := action[i+1 : j]
	args := map[string]string{}
	if inner != "" && !strings.Contains(inner, "=") {
		args["_pos"] = strings.TrimSpace(inner)
	} else {
		for _, part := range strings.Split(inner, ",") {
			if k, v, ok := strings.Cut(part, "="); ok {
				args[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
	}
	return name, args
}

func (c *crmStore) doAction(action, ruleID string, ev map[string]any, cid, aid string, pending *[]any) (map[string]any, error) {
	name, args := parseAction(action)
	switch name {
	case "create_deal_if_none":
		if cid == "" {
			return nil, nil
		}
		unlock, err := c.a.lock("deal_contact_" + c.resolve(cid))
		if err != nil {
			return nil, err
		}
		defer unlock()
		if c.openDealFor(cid) == nil {
			stage := args["stage"]
			if stage == "" {
				stage = "new_reply"
			}
			d, err := c.createDeal(cid, stage, "default_sales", "rule:"+ruleID, aid, nil)
			if err != nil {
				return nil, err
			}
			return map[string]any{"deal_id": d["id"]}, nil
		}
		return nil, nil
	case "create_task":
		title := args["title"]
		if title == "" {
			title = "Follow up"
		}
		if cid != "" && c.hasOpenTask(cid, title) {
			return nil, nil
		}
		t, err := c.addTask(title, cid, nil, dueToISO(args["due"]), "rule:"+ruleID, ruleID+":"+aid)
		if err != nil {
			return nil, err
		}
		return map[string]any{"task_id": t["id"]}, nil
	case "freeze_sequence":
		if cid != "" {
			if _, err := c.setContact(cid, map[string]any{"sequence_state": "frozen"}); err != nil {
				return nil, err
			}
		}
		return map[string]any{"frozen": true}, nil
	case "suppress", "suppress(contact)":
		reasonMap := map[string]string{"reply_negative": "reply_negative", "remove_intent": "remove_intent",
			"hard_bounce": "hard_bounce", "unsubscribe": "unsubscribe"}
		reason, ok := reasonMap[mStr(ev, "type")]
		if !ok {
			reason = "manual"
		}
		n, err := c.suppressContact(cid, reason, aid, "rule:"+ruleID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"suppressed_identities": n}, nil
	case "close_open_tasks":
		n, err := c.closeTasks(cid)
		if err != nil {
			return nil, err
		}
		return map[string]any{"closed": n}, nil
	case "set_lifecycle":
		lc := args["_pos"]
		if lc == "" {
			lc = "customer"
		}
		if cid != "" {
			if _, err := c.setContact(cid, map[string]any{"lifecycle_stage": lc}); err != nil {
				return nil, err
			}
		}
		return map[string]any{"lifecycle": lc}, nil
	case "enroll_segment":
		segName := args["_pos"]
		if segName == "" {
			segName = "customers"
		}
		if cid != "" {
			ct := c.getContact(cid)
			tagSet := map[string]bool{}
			for _, t := range mList(ct, "tags") {
				if s, ok := t.(string); ok {
					tagSet[s] = true
				}
			}
			tagSet["segment:"+segName] = true
			keys := make([]string, 0, len(tagSet))
			for k := range tagSet {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			tags := make([]any, len(keys))
			for i, k := range keys {
				tags[i] = k
			}
			if _, err := c.setContact(cid, map[string]any{"tags": tags}); err != nil {
				return nil, err
			}
		}
		return map[string]any{"segment": segName}, nil
	case "draft_reply_for_approval", "flag_in_report":
		*pending = append(*pending, map[string]any{"action": name, "contact_id": cid,
			"activity_id": aid, "rule": ruleID})
		return nil, nil
	}
	return nil, nil
}

func dueToISO(due string) string {
	if due == "" {
		return ""
	}
	if strings.HasPrefix(due, "+") && (strings.HasSuffix(due, "h") || strings.HasSuffix(due, "d")) {
		nStr := due[1 : len(due)-1]
		n := 0
		if _, err := fmt.Sscanf(nStr, "%d", &n); err != nil || fmt.Sprint(n) != nStr {
			return ""
		}
		base, err := parseISO(nowISO())
		if err != nil {
			return ""
		}
		var delta time.Duration
		if strings.HasSuffix(due, "h") {
			delta = time.Duration(n) * time.Hour
		} else {
			delta = time.Duration(n) * 24 * time.Hour
		}
		return base.Add(delta).UTC().Format("2006-01-02T15:04:05Z")
	}
	return due
}

// --- misc helpers ----------------------------------------------------------------

func (c *crmStore) clientSlug() string {
	s := filepath.Base(filepath.Dir(c.clientDir))
	if s == "" || s == "." || s == string(filepath.Separator) {
		return "client"
	}
	return s
}

// --- campaign update (operator-editable config, whitelist-merged) ----------------

var campaignOnFail = map[string]bool{"skip": true, "default_link": true}

// campaignUpdate applies an OPERATOR edit to campaign_config.json through a
// strict whitelist — goal fields, companion_doc, daily_quota, status — so a
// UI/chat patch can never rewrite sequence/guardrail/audience machinery by
// accident. Unknown keys are rejected loudly, not dropped silently.
func (c *crmStore) campaignUpdate(slug string, patch map[string]any) (map[string]any, error) {
	cfg := c.getCampaign(slug)
	if cfg == nil {
		return nil, storageErrf("campaign %q not found", slug)
	}
	changed := []string{}
	setStr := func(dst map[string]any, key, val, label string) {
		if mStr(dst, key) != val {
			dst[key] = val
			changed = append(changed, label)
		}
	}
	for key, val := range patch {
		switch key {
		case "status":
			s, _ := val.(string)
			if s != "active" && s != "paused" {
				return nil, storageErrf("status must be active|paused, got %q", s)
			}
			setStr(cfg, "status", s, "status")
		case "channel_strategy":
			s, _ := val.(string)
			if !channelStrategies[s] {
				return nil, storageErrf("channel_strategy must be one of %v, got %q", sortedChannelStrategies(), s)
			}
			setStr(cfg, "channel_strategy", s, "channel_strategy")
		case "audience.groups":
			// The groups a COMMENT campaign watches. Operator-owned and deliberately its own
			// list rather than a pointer at the client's scanned sources: reading a group for
			// leads and commenting in it carry different risk, so the operator must be able to
			// narrow this without touching what the read pipeline monitors.
			l, ok := val.([]any)
			if !ok {
				return nil, storageErrf("audience.groups must be a list of group urls, got %T", val)
			}
			groups, err := normalizeGroupURLs(l)
			if err != nil {
				return nil, err
			}
			aud := mMap(cfg, "audience")
			if aud == nil {
				aud = map[string]any{}
				cfg["audience"] = aud
			}
			before := fmt.Sprint(mList(aud, "groups"))
			if before != fmt.Sprint(groups) {
				aud["groups"] = groups
				changed = append(changed, "audience.groups")
			}
		case "seed_profiles":
			// Leads-From-Friends: the profiles whose friend lists this campaign walks.
			// One per line in the UI; each is normalized to its clean store form and
			// de-duplicated by identity so a pasted variant never becomes a second seed.
			l, ok := val.([]any)
			if !ok {
				return nil, storageErrf("seed_profiles must be a list of profile urls, got %T", val)
			}
			seeds, err := normalizeSeedProfiles(l)
			if err != nil {
				return nil, err
			}
			if fmt.Sprint(mList(cfg, "seed_profiles")) != fmt.Sprint(seeds) {
				cfg["seed_profiles"] = seeds
				changed = append(changed, "seed_profiles")
			}
		case "zillow_locations":
			l, ok := val.([]any)
			if !ok {
				return nil, storageErrf("zillow_locations must be a list of directory urls, got %T", val)
			}
			locs, err := normalizeZillowLocations(l)
			if err != nil {
				return nil, err
			}
			if fmt.Sprint(mList(cfg, "zillow_locations")) != fmt.Sprint(locs) {
				cfg["zillow_locations"] = locs
				changed = append(changed, "zillow_locations")
			}
		case "zillow_keywords":
			kws := normalizeZillowKeywords(val)
			if fmt.Sprint(mList(cfg, "zillow_keywords")) != fmt.Sprint(kws) {
				cfg["zillow_keywords"] = kws
				changed = append(changed, "zillow_keywords")
			}
		case "harvest":
			// Per-campaign harvest overrides (daily_budget, per_collector_budget, leg_pages,
			// quiet_from/to, goal_keywords). Blank fields fall back to system settings.
			hm, ok := val.(map[string]any)
			if !ok {
				return nil, storageErrf("harvest must be an object, got %T", val)
			}
			clean, err := validateHarvestBlock(hm)
			if err != nil {
				return nil, err
			}
			// MERGE, don't replace: the UI submits only the fields it shows
			// (goal_keywords + daily_budget); a per_collector_budget/leg_pages/
			// quiet override set via CLI must survive a page save. A key sent
			// explicitly as 0/"" clears that override.
			merged := map[string]any{}
			for k, v := range mMap(cfg, "harvest") {
				merged[k] = v
			}
			for k, v := range hm {
				if cv, ok := clean[k]; ok {
					merged[k] = cv
				} else if v == nil || asFloat(v, -1) == 0 || (fmt.Sprint(v) == "") {
					delete(merged, k) // explicit clear
				}
			}
			if fmt.Sprint(mMap(cfg, "harvest")) != fmt.Sprint(merged) {
				cfg["harvest"] = merged
				changed = append(changed, "harvest")
			}
		case "daily_quota":
			q := int(asFloat(val, -1))
			if q < 1 || q > 500 {
				return nil, storageErrf("daily_quota must be 1..500, got %v", val)
			}
			if mInt(cfg, "daily_quota", 0) != q {
				cfg["daily_quota"] = q
				changed = append(changed, "daily_quota")
			}
		case "sendboxes":
			// Which boxes this campaign may rotate onto. An EMPTY list means every
			// healthy box, which is usually what an operator who connected eleven of
			// them meant: a live campaign kept the single-box list it was created with
			// and queued 91 drafts onto one 20/day box while ten sat idle.
			l, ok := val.([]any)
			if !ok {
				return nil, storageErrf("sendboxes must be a list of slugs, got %T", val)
			}
			known := map[string]bool{}
			for _, b := range c.sendboxes() {
				known[mStr(b, "slug")] = true
			}
			norm := []any{}
			seen := map[string]bool{}
			for _, e := range l {
				sl := strings.TrimSpace(fmt.Sprint(e))
				if sl == "" || seen[sl] {
					continue
				}
				if !known[sl] {
					return nil, storageErrf("unknown sendbox %q — connect it on the Sendboxes page first", sl)
				}
				seen[sl] = true
				norm = append(norm, sl)
			}
			if string(marshalLineJSON(cfg["sendboxes"])) != string(marshalLineJSON(norm)) {
				cfg["sendboxes"] = norm
				changed = append(changed, "sendboxes")
			}
		case "goal":
			gp, ok := val.(map[string]any)
			if !ok {
				return nil, storageErrf("goal must be an object")
			}
			goal := mMap(cfg, "goal")
			if goal == nil {
				goal = map[string]any{}
				cfg["goal"] = goal
			}
			for gk, gv := range gp {
				switch gk {
				case "goal_type":
					s, _ := gv.(string)
					if !goalTypes[s] {
						return nil, storageErrf("goal_type %q not in %v", s, sortedGoalTypes())
					}
					setStr(goal, "goal_type", s, "goal.goal_type")
				case "description", "objective", "offer", "value_proposition":
					s, _ := gv.(string)
					setStr(goal, gk, s, "goal."+gk)
				case "proof_points":
					if l, ok := gv.([]any); ok {
						if string(marshalLineJSON(goal["proof_points"])) != string(marshalLineJSON(l)) {
							goal["proof_points"] = l
							changed = append(changed, "goal.proof_points")
						}
					}
				case "message_bank":
					// The key messages every email must land. Editable by the
					// operator because an agent-authored bank is the product
					// paraphrasing itself — which is what the live campaign had:
					// 12 entries, all source:agent, and not one of them teaching
					// anything. `source` is preserved so that stays visible.
					if l, ok := gv.([]any); ok {
						norm := make([]any, 0, len(l))
						for _, e := range l {
							em, ok := e.(map[string]any)
							if !ok {
								if s, isStr := e.(string); isStr && strings.TrimSpace(s) != "" {
									em = map[string]any{"msg": s}
								} else {
									continue
								}
							}
							msg := strings.TrimSpace(mStr(em, "msg"))
							if msg == "" {
								continue
							}
							src := mStr(em, "source")
							if src != "agent" {
								src = "operator"
							}
							ne := map[string]any{"msg": msg, "source": src,
								"approved": src == "operator" || truthy(em["approved"])}
							if en := strings.TrimSpace(mStr(em, "msg_en")); en != "" {
								ne["msg_en"] = en
							}
							norm = append(norm, ne)
						}
						if string(marshalLineJSON(goal["message_bank"])) != string(marshalLineJSON(norm)) {
							goal["message_bank"] = norm
							changed = append(changed, "goal.message_bank")
						}
					}
				case "cta":
					if cm, ok := gv.(map[string]any); ok {
						cta := mMap(goal, "cta")
						if cta == nil {
							cta = map[string]any{}
							goal["cta"] = cta
						}
						if s, ok := cm["text"].(string); ok {
							setStr(cta, "text", s, "goal.cta.text")
						}
					}
				case "companion_doc":
					if gv == nil {
						if _, had := goal["companion_doc"]; had {
							delete(goal, "companion_doc")
							changed = append(changed, "goal.companion_doc")
						}
						continue
					}
					cd, ok := gv.(map[string]any)
					if !ok {
						return nil, storageErrf("companion_doc must be an object or null")
					}
					onFail := mStr(cd, "on_fail")
					if onFail != "" && !campaignOnFail[onFail] {
						return nil, storageErrf("companion_doc.on_fail must be skip|default_link")
					}
					if onFail == "default_link" && !validEvidenceURL(cd["default_link"]) {
						return nil, storageErrf("companion_doc.default_link must be a valid http(s) URL when on_fail=default_link")
					}
					if dl := mStr(cd, "default_link"); dl != "" && !validEvidenceURL(dl) {
						return nil, storageErrf("companion_doc.default_link must be a valid http(s) URL")
					}
					newCD := map[string]any{
						"instructions": mStr(cd, "instructions"),
						"on_fail":      strOr(onFail, "skip"),
						"default_link": mStr(cd, "default_link"),
					}
					if string(marshalLineJSON(goal["companion_doc"])) != string(marshalLineJSON(newCD)) {
						goal["companion_doc"] = newCD
						changed = append(changed, "goal.companion_doc")
					}
				default:
					return nil, storageErrf("goal.%s is not operator-editable", gk)
				}
			}
		case "audience":
			// Only the PERSONALIZATION POLICY is operator-editable here (the
			// segment itself is campaign design, not a knob): whether a lead with
			// no proof-of-life may still be emailed, and the confidence floor.
			ap, ok := val.(map[string]any)
			if !ok {
				return nil, storageErrf("audience must be an object")
			}
			for ak, av := range ap {
				if ak != "personalization" {
					return nil, storageErrf("audience.%s is not operator-editable (only audience.personalization)", ak)
				}
				pm, ok := av.(map[string]any)
				if !ok {
					return nil, storageErrf("audience.personalization must be an object")
				}
				aud := mMap(cfg, "audience")
				if aud == nil {
					aud = map[string]any{}
					cfg["audience"] = aud
				}
				pers := mMap(aud, "personalization")
				if pers == nil {
					pers = map[string]any{}
					aud["personalization"] = pers
				}
				for pk, pv := range pm {
					switch pk {
					case "no_hook_fallback":
						s, _ := pv.(string)
						if s != "skip" && s != "generic_honest_opener" {
							return nil, storageErrf("no_hook_fallback must be skip|generic_honest_opener, got %q", s)
						}
						setStr(pers, "no_hook_fallback", s, "audience.personalization.no_hook_fallback")
					case "min_confidence":
						f := asFloat(pv, -1)
						if f < 0 || f > 1 {
							return nil, storageErrf("min_confidence must be 0..1, got %v", pv)
						}
						if asFloat(pers["min_confidence"], -1) != f {
							pers["min_confidence"] = f
							changed = append(changed, "audience.personalization.min_confidence")
						}
					default:
						return nil, storageErrf("audience.personalization.%s is not operator-editable", pk)
					}
				}
			}
		default:
			return nil, storageErrf("%s is not operator-editable (allowed: status, daily_quota, goal.*, audience.personalization.*)", key)
		}
	}
	p, err := c.campaignConfigPath(slug)
	if err != nil {
		return nil, err
	}
	if len(changed) > 0 {
		if err := atomicWriteFile(p, marshalIndentJSON(cfg)); err != nil {
			return nil, err
		}
	}
	return map[string]any{"ok": true, "campaign": slug, "changed": strsToAny(changed), "config": cfg}, nil
}

// pyTitle mirrors Python str.title(): letter after non-letter uppercased, rest lowered.
func pyTitle(s string) string {
	var sb strings.Builder
	prevLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			if prevLetter {
				sb.WriteRune(unicode.ToLower(r))
			} else {
				sb.WriteRune(unicode.ToUpper(r))
			}
			prevLetter = true
		} else {
			sb.WriteRune(r)
			prevLetter = false
		}
	}
	return sb.String()
}

func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func monthEndExclusive(month string) string {
	y, m := 0, 0
	fmt.Sscanf(month[:4], "%d", &y)
	fmt.Sscanf(month[5:7], "%d", &m)
	if m == 12 {
		y, m = y+1, 1
	} else {
		m++
	}
	return fmt.Sprintf("%04d-%02d-01T00:00:00Z", y, m)
}
