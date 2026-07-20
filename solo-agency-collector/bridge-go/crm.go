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

func (c *crmStore) createCampaign(slug string, config map[string]any) (map[string]any, error) {
	goal := mMap(config, "goal")
	gt := mStr(goal, "goal_type")
	if gt != "" && !goalTypes[gt] {
		return nil, storageErrf("goal_type %q not in %v", gt, sortedGoalTypes())
	}
	if gt == "" {
		gt = "get_reply"
	}
	cfg := map[string]any{
		"schema_version": 1, "campaign_slug": slug,
		"goal": map[string]any{"goal_type": gt, "objective": "", "offer": "",
			"value_proposition": "", "proof_points": []any{},
			"cta":           map[string]any{"type": "reply_yes", "text": ""},
			"success_event": map[string]any{"on": "reply_positive", "create_deal_stage": "new_reply"}},
		"audience": map[string]any{"segment": "", "personalization": map[string]any{
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
		"in_active_sequence": 0, "no_email": 0}
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
	identityTTLDays = 90
	hookTTLDays     = 10
	negRetryDays    = 30
)

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
	hooksFresh := mStr(en, "hooks_refreshed_at") != "" && mStr(en, "hooks_refreshed_at") >= isoDaysAgoFrom(hookTTLDays, now)
	nf := mStr(en, "email_not_found_at")
	if mStr(ident, "still_active") == "inactive" && identFresh {
		return map[string]any{"needs": "skip", "reason": "known_inactive"}
	}
	if !identFresh {
		return map[string]any{"needs": "enrich", "reason": "identity_stale_or_missing"}
	}
	if nf != "" && nf >= isoDaysAgoFrom(negRetryDays, now) {
		return map[string]any{"needs": "skip", "reason": "email_not_found_recent"}
	}
	if !hooksFresh {
		return map[string]any{"needs": "refresh", "reason": "hooks_stale"}
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
	hasIdentity := len(ident) > 0
	mergedIdent := map[string]any{}
	for _, k := range []string{"still_active", "current_company", "role", "profiles", "evidence"} {
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
		enrichment["email_not_found_at"] = now
	}
	if len(usableHooks) == 0 && truthy(dossier["mark_no_hook"]) {
		enrichment["no_verifiable_hook_at"] = now
	}
	patch := map[string]any{"enrichment": enrichment}
	found := mMap(ident, "channels_found")
	var emails, phones []any
	for _, e := range mList(found, "emails") {
		if s, ok := e.(string); ok && s != "" {
			emails = append(emails, map[string]any{"address": s, "source": "enrich", "status": "unverified"})
		}
	}
	for _, p := range mList(found, "phones") {
		if s, ok := p.(string); ok && s != "" {
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
				} else {
					foundSocials[platform] = s
				}
			}
		}
	case []any:
		for _, u := range pv {
			if s, ok := u.(string); ok && s != "" {
				kind, platform := classifyLeadURL(s)
				switch kind {
				case "profile":
					foundSocials[platform] = s
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
	return map[string]any{"lead_id": leadID, "usable_hooks": len(usableHooks), "confidence_band": band,
		"do_not_mention": len(doNotMention), "problems": problems}, nil
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
			return c.resolve(existing), "matched", nil
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

func (c *crmStore) setContact(leadID string, patch map[string]any) (map[string]any, error) {
	leadID = c.resolve(leadID)
	rec, err := c.a.update("contacts", leadID, func(rec map[string]any) map[string]any {
		mergeIntoContact(rec, patch)
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
