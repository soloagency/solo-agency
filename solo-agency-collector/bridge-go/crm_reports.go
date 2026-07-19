package main

// crm_reports.go — crm_store.py port, part 2: sendbox rotation, draft
// validation + storage, approval report/apply, ingest-ui, follow-ups, Today
// View / kanban / weekly / monthly reports. Rendering goes through
// renderReportFile (in-binary renderer; report_renderer.py's subprocess sites
// disappear here, which also removes the hardcoded `python3` that broke Windows).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// --- sendboxes -------------------------------------------------------------------

func (c *crmStore) sendboxes() []map[string]any {
	p := filepath.Join(c.clientDir, "sendboxes", "sendboxes.json")
	if m, err := readJSONFile(p); err == nil {
		return mapsOf(mList(m, "sendboxes"))
	}
	return nil
}

func (c *crmStore) sentToday(sendboxSlug, day string) int {
	n := 0
	for _, p := range c.allSentLogs("") {
		for _, r := range readJSONLines(p) {
			sa := mStr(r, "sent_at")
			if len(sa) >= 10 {
				sa = sa[:10]
			}
			if mStr(r, "sendbox") == sendboxSlug && sa == day && mStr(r, "rfc_message_id") != "" {
				n++
			}
		}
	}
	return n
}

func (c *crmStore) pickSendbox(campaignCfg, contact map[string]any, day string) string {
	if s := mStr(contact, "assigned_sendbox"); s != "" {
		return s
	}
	if day == "" {
		day = todayStr("")
	}
	refs := map[string]bool{}
	for _, r := range mList(campaignCfg, "sendboxes") {
		if s, ok := r.(string); ok {
			refs[s] = true
		}
	}
	var boxes []map[string]any
	for _, b := range c.sendboxes() {
		if mStr(b, "status") != "healthy" || mInt(b, "quota_today", 0) <= 0 {
			continue
		}
		if len(refs) > 0 && !refs[mStr(b, "slug")] {
			continue
		}
		boxes = append(boxes, b)
	}
	if len(boxes) == 0 {
		return ""
	}
	load := func(b map[string]any) float64 {
		q := mInt(b, "quota_today", 1)
		if q == 0 {
			q = 1
		}
		return float64(c.sentToday(mStr(b, "slug"), day)) / float64(q)
	}
	sort.SliceStable(boxes, func(i, j int) bool {
		li, lj := load(boxes[i]), load(boxes[j])
		if li != lj {
			return li < lj
		}
		return mStr(boxes[i], "slug") < mStr(boxes[j], "slug")
	})
	return mStr(boxes[0], "slug")
}

// --- drafts ----------------------------------------------------------------------

// subjectGateNormalized: NFKC-fold the characters that matter for the Re:/Fwd:
// gate (fullwidth ASCII + ideographic space) and strip Unicode Cf (format)
// chars — same bypasses adapter.py's unicodedata pass closes (zero-width space,
// fullwidth colon). Stdlib-only stand-in for full NFKC.
func subjectGateNormalized(subject string) string {
	var sb strings.Builder
	for _, r := range subject {
		switch {
		case r >= 0xFF01 && r <= 0xFF5E: // fullwidth ASCII block -> ASCII
			r = r - 0xFF00 + 0x20
		case r == 0x3000: // ideographic space
			r = ' '
		}
		if unicode.Is(unicode.Cf, r) {
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

var reFwdRe = regexp.MustCompile(`(?i)^\s*(re|fwd)\s*:`)

type draftArgs struct {
	Step             int
	Subject          string
	BodyText         string
	BodyHTML         string
	HooksUsed        []map[string]any
	Tracking         string
	IsReply          bool
	BankMessagesUsed []any
	CompanionURL     string
}

func (c *crmStore) draftWrite(contactID, campaignSlug string, a draftArgs) (map[string]any, error) {
	leadID := c.resolve(contactID)
	contact := c.getContact(leadID)
	if contact == nil {
		return nil, storageErrf("contact_not_found")
	}
	cfg := c.getCampaign(campaignSlug)
	if cfg == nil {
		return nil, storageErrf("campaign %q not found", campaignSlug)
	}
	step := a.Step
	if !a.IsReply {
		budget, err := c.draftBudget(campaignSlug, "")
		if err != nil {
			return nil, err
		}
		remaining := mInt(budget, "remaining", 0)
		if remaining <= 0 {
			return nil, storageErrf("draft_budget_exhausted: today's daily_quota draft slots are used")
		}
		if step > 1 {
			dq := mInt(cfg, "daily_quota", 40)
			floor := dq / 5
			if floor < 1 {
				floor = 1
			}
			if v, ok := cfg["new_lead_floor"]; ok {
				floor = int(asFloat(v, float64(floor)))
			}
			if remaining <= floor {
				return nil, storageErrf("bump_budget_exhausted: remaining %d draft slots are reserved for new-lead (step-1) drafts (floor %d)", remaining, floor)
			}
		}
	}
	if step > 1 {
		open, err := c.openDraftSteps(campaignSlug)
		if err != nil {
			return nil, err
		}
		if open[leadStep{leadID, step}] {
			return nil, storageErrf("duplicate_pending_draft: a draft for this lead/step already awaits approval")
		}
	}
	if step == 1 && reFwdRe.MatchString(subjectGateNormalized(a.Subject)) {
		return nil, storageErrf("step-1 subject must not begin with Re:/Fwd: (deceptive, CAN-SPAM)")
	}
	var primary map[string]any
	var emails []map[string]any
	for _, e := range mapsOf(mList(mMap(contact, "identities"), "emails")) {
		if validEmail(mStr(e, "address")) {
			emails = append(emails, e)
		}
	}
	for _, e := range emails {
		if mBool(e, "is_primary") {
			primary = e
			break
		}
	}
	if primary == nil && len(emails) > 0 {
		primary = emails[0]
	}
	if primary == nil {
		return nil, storageErrf("contact has no usable email to draft to")
	}
	dossierHooks := mapsOf(mList(mMap(contact, "enrichment"), "hooks"))
	evidenceByURL := map[string]map[string]any{}
	for _, h := range dossierHooks {
		if validEvidenceURL(h["evidence_url"]) {
			evidenceByURL[mStr(h, "evidence_url")] = h
		}
	}
	var cleanHooks []any
	for _, h := range a.HooksUsed {
		url := mStr(h, "evidence_url")
		dh, ok := evidenceByURL[url]
		if url == "" || !ok {
			return nil, storageErrf("hook %s has no matching evidenced dossier hook — every personalized detail must trace to a dossier hook with an evidence_url", pyRepr(hookType(h)))
		}
		cleanHooks = append(cleanHooks, map[string]any{"type": dh["type"], "evidence_url": url})
	}
	if step == 1 && len(cleanHooks) == 0 {
		fallback := mStr(mMap(mMap(cfg, "audience"), "personalization"), "no_hook_fallback")
		if fallback == "" {
			fallback = "skip"
		}
		if fallback != "generic_honest_opener" {
			return nil, storageErrf("no_evidenced_hook: this campaign requires a recent evidenced hook (proof-of-life) for a step-1 email; no_hook_fallback=skip")
		}
	}
	sendbox := c.pickSendbox(cfg, contact, "")
	if sendbox == "" {
		return nil, storageErrf("no healthy sendbox available for this campaign")
	}
	band := mStr(mMap(contact, "enrichment"), "confidence_band")
	if band == "" {
		band = "review_carefully"
	}
	warnings := []any{}
	if len(cleanHooks) == 0 {
		warnings = append(warnings, "generic_opener")
	}
	if step > 1 {
		warnings = append(warnings, "bump_step")
	}
	did := newULID("draft_")
	now := nowISO()
	if a.Tracking == "" {
		a.Tracking = "plain_text"
	}
	if a.BankMessagesUsed == nil {
		a.BankMessagesUsed = []any{}
	}
	if cleanHooks == nil {
		cleanHooks = []any{}
	}
	draft := map[string]any{"id": did, "schema_version": 1, "created_at": now, "updated_at": now,
		"lead_id": leadID, "campaign_slug": campaignSlug, "step": step,
		"sendbox": sendbox, "to": mStr(primary, "address"), "subject": a.Subject,
		"body_text": a.BodyText, "body_html": a.BodyHTML,
		"confidence_band": band, "hooks_used": cleanHooks, "tracking": a.Tracking,
		"warnings": warnings, "guessed_approved": false, "is_reply": a.IsReply,
		"bank_messages_used": a.BankMessagesUsed, "companion_url": a.CompanionURL,
		"status": "pending_approval", "decided_at": "", "decided_by": "", "reject_reason": "", "blocker": ""}
	cd, err := c.campaignDir(campaignSlug)
	if err != nil {
		return nil, err
	}
	d := filepath.Join(cd, "outbox", "pending_approval", todayStr(now))
	if err := os.MkdirAll(d, 0o755); err != nil {
		return nil, err
	}
	draftPath := filepath.Join(d, did+".json")
	if err := atomicWriteFile(draftPath, marshalIndentJSON(draft)); err != nil {
		return nil, err
	}
	if len(cleanHooks) > 0 {
		usedURLs := map[string]bool{}
		for _, h := range mapsOf(cleanHooks) {
			usedURLs[mStr(h, "evidence_url")] = true
		}
		tag := fmt.Sprintf("%s/step%d", campaignSlug, step)
		if _, err := c.a.update("contacts", leadID, func(rec map[string]any) map[string]any {
			for _, hk := range mapsOf(mList(mMap(rec, "enrichment"), "hooks")) {
				if usedURLs[mStr(hk, "evidence_url")] {
					ui := mList(hk, "used_in")
					seen := false
					for _, x := range ui {
						if s, ok := x.(string); ok && s == tag {
							seen = true
							break
						}
					}
					if !seen {
						hk["used_in"] = append(ui, tag)
					}
				}
			}
			return rec
		}); err != nil {
			return nil, err
		}
	}
	return map[string]any{"draft_id": did, "sendbox": sendbox, "to": mStr(primary, "address"),
		"confidence_band": band, "warnings": warnings, "path": draftPath}, nil
}

func (c *crmStore) listPendingDrafts(campaignSlug string) []map[string]any {
	out := []map[string]any{}
	campRoot := filepath.Join(c.clientDir, "campaigns")
	var camps []string
	if campaignSlug != "" {
		camps = []string{campaignSlug}
	} else {
		entries, _ := os.ReadDir(campRoot)
		for _, e := range entries {
			camps = append(camps, e.Name())
		}
		sort.Strings(camps)
	}
	for _, camp := range camps {
		base := filepath.Join(campRoot, camp, "outbox", "pending_approval")
		days, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		dayNames := make([]string, 0, len(days))
		for _, d := range days {
			dayNames = append(dayNames, d.Name())
		}
		sort.Strings(dayNames)
		for _, day := range dayNames {
			dd := filepath.Join(base, day)
			if st, err := os.Stat(dd); err != nil || !st.IsDir() {
				continue
			}
			files, _ := os.ReadDir(dd)
			fnames := make([]string, 0, len(files))
			for _, f := range files {
				fnames = append(fnames, f.Name())
			}
			sort.Strings(fnames)
			for _, name := range fnames {
				if !strings.HasSuffix(name, ".json") {
					continue
				}
				p := filepath.Join(dd, name)
				rec, err := readJSONFile(p)
				if err != nil {
					continue
				}
				if mStr(rec, "status") == "pending_approval" {
					rec["_path"] = p
					out = append(out, rec)
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if mStr(a, "campaign_slug") != mStr(b, "campaign_slug") {
			return mStr(a, "campaign_slug") < mStr(b, "campaign_slug")
		}
		if mStr(a, "created_at") != mStr(b, "created_at") {
			return mStr(a, "created_at") < mStr(b, "created_at")
		}
		return mStr(a, "id") < mStr(b, "id")
	})
	return out
}

func (c *crmStore) draftBudget(campaignSlug, day string) (map[string]any, error) {
	cfg := c.getCampaign(campaignSlug)
	if cfg == nil {
		return nil, storageErrf("campaign %q not found", campaignSlug)
	}
	if day == "" {
		day = todayStr("")
	}
	dailyQuota := mInt(cfg, "daily_quota", 40)
	cd, err := c.campaignDir(campaignSlug)
	if err != nil {
		return nil, err
	}
	base := filepath.Join(cd, "outbox")
	used := 0
	for _, scan := range []string{filepath.Join(base, "pending_approval"), filepath.Join(base, "approved")} {
		_ = filepath.WalkDir(scan, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
				return nil
			}
			rec, err := readJSONFile(p)
			if err != nil {
				return nil
			}
			ca := mStr(rec, "created_at")
			if len(ca) >= 10 && ca[:10] == day {
				used++
			}
			return nil
		})
	}
	remaining := dailyQuota - used
	if remaining < 0 {
		remaining = 0
	}
	return map[string]any{"campaign": campaignSlug, "daily_quota": dailyQuota,
		"used_today": used, "remaining": remaining}, nil
}

type leadStep struct {
	Lead string
	Step int
}

func (c *crmStore) openDraftSteps(campaignSlug string) (map[leadStep]bool, error) {
	out := map[leadStep]bool{}
	cd, err := c.campaignDir(campaignSlug)
	if err != nil {
		return nil, err
	}
	base := filepath.Join(cd, "outbox")
	for _, sub := range []string{"pending_approval", "approved"} {
		_ = filepath.WalkDir(filepath.Join(base, sub), func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
				return nil
			}
			rec, err := readJSONFile(p)
			if err != nil {
				return nil
			}
			st := mStr(rec, "status")
			if st == "pending_approval" || st == "approved" {
				out[leadStep{c.resolve(mStr(rec, "lead_id")), mInt(rec, "step", 0)}] = true
			}
			return nil
		})
	}
	return out, nil
}

// --- Approval Report + chat approval ----------------------------------------------

func (c *crmStore) buildApproval(campaignSlug, now string, numberByDraft map[string]int) (string, []map[string]any) {
	if now == "" {
		now = nowISO()
	}
	drafts := c.listPendingDrafts(campaignSlug)
	used := map[int]bool{}
	maxN := 0
	for _, n := range numberByDraft {
		used[n] = true
		if n > maxN {
			maxN = n
		}
	}
	nxt := maxN + 1
	var nums []numberedDraft
	for _, d := range drafts {
		n, ok := numberByDraft[mStr(d, "id")]
		if !ok {
			n = nxt
			nxt++
		}
		nums = append(nums, numberedDraft{n, d})
	}
	sort.SliceStable(nums, func(i, j int) bool { return nums[i].N < nums[j].N })
	index := make([]map[string]any, 0, len(nums))
	for _, nd := range nums {
		index = append(index, map[string]any{"n": nd.N, "draft_id": mStr(nd.D, "id"),
			"path": mStr(nd.D, "_path"), "campaign": mStr(nd.D, "campaign_slug")})
	}
	var newOnes, followups, high, review []numberedDraft
	for _, nd := range nums {
		if mInt(nd.D, "step", 1) == 1 {
			newOnes = append(newOnes, nd)
		} else {
			followups = append(followups, nd)
		}
	}
	for _, nd := range newOnes {
		if mStr(nd.D, "confidence_band") == "high" {
			high = append(high, nd)
		} else {
			review = append(review, nd)
		}
	}
	card := func(n int, d map[string]any) string {
		ct := c.getContact(mStr(d, "lead_id"))
		en := mMap(ct, "enrichment")
		name := mStr(mMap(ct, "name"), "full")
		if name == "" {
			name = mStr(d, "to")
		}
		lines := []string{fmt.Sprintf("## %d. %s — %s", n, name, mStr(d, "to")),
			fmt.Sprintf("- **Campaign/step:** %s / step %v  ·  **Sendbox:** %s",
				mStr(d, "campaign_slug"), pyNum(d["step"]), mStr(d, "sendbox"))}
		if ws := mList(d, "warnings"); len(ws) > 0 {
			strs := make([]string, len(ws))
			for i, w := range ws {
				strs[i] = fmt.Sprint(w)
			}
			lines = append(lines, fmt.Sprintf("- **Flags:** %s", strings.Join(strs, ", ")))
		}
		hooks := mapsOf(mList(en, "hooks"))
		if len(hooks) > 0 {
			var parts []string
			for _, h := range hooks {
				if u := mStr(h, "evidence_url"); u != "" {
					t := mStr(h, "type")
					if t == "" {
						t = "hook"
					}
					parts = append(parts, fmt.Sprintf("[%s](%s)", t, u))
				}
			}
			lines = append(lines, "- **Evidence:** "+strings.Join(parts, "  ·  "))
		}
		lines = append(lines, "", fmt.Sprintf("**Subject:** %s", mStr(d, "subject")), "",
			"> "+strings.ReplaceAll(mStr(d, "body_text"), "\n", "\n> "), "")
		return strings.Join(lines, "\n")
	}
	md := []string{fmt.Sprintf("# Approval Report — %s", now[:10]),
		fmt.Sprintf("%d draft(s) awaiting your approval. Reply in chat: "+
			"`approve all` · `approve 1-20, 35` · `reject 7: reason` · `edit 12: ...` · `hold 5`.", len(drafts)),
		"", fmt.Sprintf("## High confidence (%d)", len(high)),
		"*(verified email + strong evidenced hook)*", ""}
	md = appendCardsOrNone(md, high, card)
	md = append(md, "", fmt.Sprintf("## Review carefully (%d)", len(review)),
		"*(weak/no hook or fallback opener — read before approving)*", "")
	md = appendCardsOrNone(md, review, card)
	md = append(md, "", fmt.Sprintf("## Follow-ups due (%d)", len(followups)),
		"*(bumps and reply drafts — threaded onto an existing conversation)*", "")
	md = appendCardsOrNone(md, followups, card)
	return strings.Join(md, "\n"), index
}

type numberedDraft struct {
	N int
	D map[string]any
}

func appendCardsOrNone(md []string, items []numberedDraft, card func(int, map[string]any) string) []string {
	if len(items) == 0 {
		return append(md, "*(none)*", "")
	}
	for _, nd := range items {
		md = append(md, card(nd.N, nd.D))
	}
	return md
}

// pyNum renders a JSON number like Python prints it (1 not 1.0 for integral).
func pyNum(v any) string {
	if f, ok := toFloat(v); ok {
		if f == float64(int64(f)) {
			return strconv.FormatInt(int64(f), 10)
		}
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	return fmt.Sprint(v)
}

func (c *crmStore) renderApprovalReport(campaignSlug string) (map[string]any, error) {
	now := nowISO()
	prior := map[string]int{}
	for _, e := range c.approvalIndex() {
		prior[mStr(e, "draft_id")] = mInt(e, "n", 0)
	}
	md, index := c.buildApproval(campaignSlug, now, prior)
	outDir := filepath.Join(c.clientDir, "outputs", todayStr(now))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	slug := c.clientSlug()
	mdPath := filepath.Join(outDir, slug+"-approval-report.md")
	idxPath := filepath.Join(outDir, "approval_index.json")
	htmlPath := filepath.Join(outDir, slug+"-approval-report.html")
	if err := atomicWriteFile(mdPath, md); err != nil {
		return nil, err
	}
	idx := make([]any, len(index))
	for i, e := range index {
		idx[i] = e
	}
	if err := atomicWriteFile(idxPath, marshalIndentJSON(map[string]any{"generated_at": now, "index": idx})); err != nil {
		return nil, err
	}
	res := renderReportFile(rendererRequest{Input: mdPath, OutputHTML: htmlPath,
		Title: "Approval Report", ReportKind: "Approval Report"})
	out := map[string]any{"drafts": len(index), "md": mdPath, "index": idxPath,
		"html": nil, "html_rendered": res.RC == 0}
	if res.RC == 0 {
		out["html"] = htmlPath
	}
	return out, nil
}

func (c *crmStore) approvalIndex() []map[string]any {
	p := filepath.Join(c.clientDir, "outputs", todayStr(""), "approval_index.json")
	if m, err := readJSONFile(p); err == nil {
		return mapsOf(mList(m, "index"))
	}
	return nil
}

// resolveNumbers: "all" | "1-20, 35" | [1,3,5] -> index entries.
func (c *crmStore) resolveNumbers(spec any) ([]map[string]any, error) {
	idx := c.approvalIndex()
	if s, ok := spec.(string); ok && s == "all" {
		return idx, nil
	}
	byN := map[int]map[string]any{}
	for _, e := range idx {
		byN[mInt(e, "n", 0)] = e
	}
	nums, err := requestedNumbers(spec, true)
	if err != nil {
		return nil, err
	}
	sorted := make([]int, 0, len(nums))
	for n := range nums {
		sorted = append(sorted, n)
	}
	sort.Ints(sorted)
	var out []map[string]any
	for _, n := range sorted {
		if e, ok := byN[n]; ok {
			out = append(out, e)
		}
	}
	return out, nil
}

// requestedNumbers parses a number spec; strict=true raises on an over-wide range.
func requestedNumbers(spec any, strict bool) (map[int]bool, error) {
	nums := map[int]bool{}
	switch s := spec.(type) {
	case string:
		for _, part := range strings.Split(strings.ReplaceAll(s, " ", ""), ",") {
			if part == "" {
				continue
			}
			trimmed := strings.TrimLeft(part, "-")
			if strings.Contains(trimmed, "-") {
				cut := strings.Index(part[strings.Index(part, trimmed):], "-") + strings.Index(part, trimmed)
				aStr, bStr := part[:cut], part[cut+1:]
				a, err1 := strconv.Atoi(aStr)
				b, err2 := strconv.Atoi(bStr)
				if err1 != nil || err2 != nil {
					return nil, storageErrf("bad number %q", part)
				}
				if a > b {
					a, b = b, a
				}
				if b-a > 100000 {
					if strict {
						return nil, storageErrf("range %s too wide (max span 100000)", pyRepr(part))
					}
					continue
				}
				for n := a; n <= b; n++ {
					nums[n] = true
				}
			} else {
				n, err := strconv.Atoi(part)
				if err != nil {
					return nil, storageErrf("bad number %q", part)
				}
				nums[n] = true
			}
		}
	case []any:
		for _, x := range s {
			if f, ok := toFloat(x); ok {
				nums[int(f)] = true
			}
		}
	}
	return nums, nil
}

func (c *crmStore) approveApply(actions map[string]any, by string) (map[string]any, error) {
	result := map[string]any{"approved": []any{}, "rejected": []any{}, "held": []any{},
		"edited": []any{}, "not_found": []any{}, "already_processed": []any{}}
	push := func(key string, v any) { result[key] = append(result[key].([]any), v) }
	now := nowISO()
	decided := map[int]bool{}

	load := func(path string) map[string]any {
		d, err := readJSONFile(path)
		if err != nil {
			return nil
		}
		return d
	}

	for _, e := range mapsOf(mList(actions, "edit")) {
		n := mInt(e, "n", 0)
		hits, err := c.resolveNumbers([]any{float64(n)})
		if err != nil {
			return nil, err
		}
		if len(hits) == 0 {
			push("not_found", n)
			continue
		}
		entry := hits[0]
		d := load(mStr(entry, "path"))
		if d == nil {
			push("not_found", n)
			continue
		}
		if v, ok := e["subject"]; ok {
			d["subject"] = v
		}
		if v, ok := e["body_text"]; ok {
			d["body_text"] = v
		}
		d["updated_at"] = now
		if err := atomicWriteFile(mStr(entry, "path"), marshalIndentJSON(d)); err != nil {
			return nil, err
		}
		push("edited", entry["draft_id"])
	}
	for _, r := range mapsOf(mList(actions, "reject")) {
		n := mInt(r, "n", 0)
		hits, err := c.resolveNumbers([]any{float64(n)})
		if err != nil {
			return nil, err
		}
		if len(hits) == 0 {
			push("not_found", n)
			continue
		}
		if decided[n] {
			continue
		}
		entry := hits[0]
		d := load(mStr(entry, "path"))
		if d == nil || mStr(d, "status") != "pending_approval" {
			push("already_processed", entry["draft_id"])
			decided[n] = true
			continue
		}
		reason := mStr(r, "reason")
		d["status"] = "rejected"
		d["decided_at"] = now
		d["decided_by"] = by
		d["reject_reason"] = reason
		if err := atomicWriteFile(mStr(entry, "path"), marshalIndentJSON(d)); err != nil {
			return nil, err
		}
		if err := c.approvalLog(d, "reject", by, reason); err != nil {
			return nil, err
		}
		if err := c.learningLog(d, reason); err != nil {
			return nil, err
		}
		push("rejected", entry["draft_id"])
		decided[n] = true
	}
	holdHits, err := c.resolveNumbers(anyOr(actions["hold"], []any{}))
	if err != nil {
		return nil, err
	}
	for _, h := range holdHits {
		n := mInt(h, "n", 0)
		if decided[n] {
			continue
		}
		d := load(mStr(h, "path"))
		if d == nil || mStr(d, "status") != "pending_approval" {
			push("already_processed", h["draft_id"])
			decided[n] = true
			continue
		}
		d["status"] = "hold"
		d["decided_at"] = now
		d["decided_by"] = by
		if err := atomicWriteFile(mStr(h, "path"), marshalIndentJSON(d)); err != nil {
			return nil, err
		}
		push("held", h["draft_id"])
		decided[n] = true
	}
	approveSpec := actions["approve"]
	if truthy(approveSpec) {
		hits, err := c.resolveNumbers(approveSpec)
		if err != nil {
			return nil, err
		}
		if s, ok := approveSpec.(string); !ok || s != "all" {
			byN := map[int]bool{}
			for _, e := range c.approvalIndex() {
				byN[mInt(e, "n", 0)] = true
			}
			reqNums, err := requestedNumbers(approveSpec, false)
			if err != nil {
				return nil, err
			}
			sortedReq := make([]int, 0, len(reqNums))
			for n := range reqNums {
				sortedReq = append(sortedReq, n)
			}
			sort.Ints(sortedReq)
			for _, n := range sortedReq {
				if !byN[n] {
					push("not_found", n)
				}
			}
		}
		for _, a := range hits {
			n := mInt(a, "n", 0)
			if decided[n] {
				continue
			}
			d := load(mStr(a, "path"))
			if d == nil || mStr(d, "status") != "pending_approval" {
				push("already_processed", a["draft_id"])
				decided[n] = true
				continue
			}
			d["status"] = "approved"
			d["decided_at"] = now
			d["decided_by"] = by
			cd, err := c.campaignDir(mStr(a, "campaign"))
			if err != nil {
				return nil, err
			}
			approvedDir := filepath.Join(cd, "outbox", "approved")
			if err := os.MkdirAll(approvedDir, 0o755); err != nil {
				return nil, err
			}
			dest := filepath.Join(approvedDir, mStr(a, "draft_id")+".json")
			if err := atomicWriteFile(mStr(a, "path"), marshalIndentJSON(d)); err != nil {
				return nil, err
			}
			if err := os.Rename(mStr(a, "path"), dest); err != nil {
				return nil, err
			}
			if err := c.approvalLog(d, "approve", by, ""); err != nil {
				return nil, err
			}
			decided[n] = true
			push("approved", map[string]any{"draft_id": a["draft_id"], "path": dest})
		}
	}
	return result, nil
}

func anyOr(v any, def any) any {
	if v == nil {
		return def
	}
	return v
}

func (c *crmStore) approvalLog(draft map[string]any, decision, by, reason string) error {
	p := filepath.Join(c.clientDir, "approvals", "approval_log.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	_, statErr := os.Stat(p)
	isNew := statErr != nil
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if isNew {
		if _, err := f.WriteString("# Approval Log\n\n| Date | Draft | Campaign/Step | Decision | By | Reason |\n|---|---|---|---|---|---|\n"); err != nil {
			return err
		}
	}
	r := reason
	if r == "" {
		r = "—"
	}
	_, err = f.WriteString(fmt.Sprintf("| %s | %s | %s/%s | %s | %s | %s |\n",
		nowISO(), mStr(draft, "id"), mStr(draft, "campaign_slug"), pyNum(draft["step"]), decision, by, r))
	return err
}

func (c *crmStore) learningLog(draft map[string]any, reason string) error {
	if reason == "" {
		return nil
	}
	p := filepath.Join(c.clientDir, "analytics", "learning_log.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	_, statErr := os.Stat(p)
	isNew := statErr != nil
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if isNew {
		if _, err := f.WriteString("# Learning Log\n\n| Date | Source | Signal | Note |\n|---|---|---|---|\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString(fmt.Sprintf("| %s | draft_rejected | %s/step%s | %s |\n",
		nowISO(), mStr(draft, "campaign_slug"), pyNum(draft["step"]), reason))
	return err
}

// --- ingest-ui (browser approval decisions) ---------------------------------------

func (c *crmStore) ingestUIDecisions() (map[string]any, error) {
	inbox := filepath.Join(c.clientDir, "ui_inbox", "approval_decisions.jsonl")
	cursorPath := filepath.Join(c.clientDir, "ui_inbox", ".approval_cursor")
	result := map[string]any{"approved": []any{}, "rejected": []any{}, "held": []any{},
		"edited": []any{}, "not_found": []any{}, "already_processed": []any{}, "processed_lines": 0}
	push := func(key string, v any) { result[key] = append(result[key].([]any), v) }
	data, err := os.ReadFile(inbox)
	if err != nil {
		return result, nil
	}
	cursor := 0
	if cdata, err := os.ReadFile(cursorPath); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(cdata))); err == nil {
			cursor = n
		}
	}
	lines := strings.SplitAfter(string(data), "\n")
	// SplitAfter leaves a trailing "" when the file ends with \n — drop it to match readlines()
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	now := nowISO()

	findPending := func(draftID, campaign string) (string, string) {
		var camps []string
		if campaign != "" {
			camps = []string{campaign}
		} else {
			base := filepath.Join(c.clientDir, "campaigns")
			entries, _ := os.ReadDir(base)
			for _, e := range entries {
				camps = append(camps, e.Name())
			}
		}
		for _, camp := range camps {
			var hit string
			base := filepath.Join(c.clientDir, "campaigns", camp, "outbox", "pending_approval")
			_ = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if d.Name() == draftID+".json" && hit == "" {
					hit = p
				}
				return nil
			})
			if hit != "" {
				return camp, hit
			}
		}
		return "", ""
	}

	if cursor > len(lines) {
		cursor = len(lines)
	}
	for _, raw := range lines[cursor:] {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		var dec map[string]any
		if json.Unmarshal([]byte(raw), &dec) != nil {
			continue
		}
		did := mStr(dec, "draft_id")
		action := mStr(dec, "decision")
		camp, path := findPending(did, mStr(dec, "campaign"))
		if path == "" {
			push("not_found", did)
			continue
		}
		d, err := readJSONFile(path)
		if err != nil {
			push("not_found", did)
			continue
		}
		if mStr(d, "status") != "pending_approval" {
			push("already_processed", did)
			continue
		}
		if es := mStr(dec, "edited_subject"); es != "" {
			d["subject"] = es
		}
		if eb := mStr(dec, "edited_body"); eb != "" {
			d["body_text"] = eb
		}
		d["updated_at"] = now
		switch action {
		case "edit":
			if err := atomicWriteFile(path, marshalIndentJSON(d)); err != nil {
				return nil, err
			}
			push("edited", did)
		case "reject":
			reason := mStr(dec, "note")
			d["status"] = "rejected"
			d["decided_at"] = now
			d["decided_by"] = "ui"
			d["reject_reason"] = reason
			if err := atomicWriteFile(path, marshalIndentJSON(d)); err != nil {
				return nil, err
			}
			if err := c.approvalLog(d, "reject", "ui", reason); err != nil {
				return nil, err
			}
			if err := c.learningLog(d, reason); err != nil {
				return nil, err
			}
			push("rejected", did)
		case "hold":
			d["status"] = "hold"
			d["decided_at"] = now
			d["decided_by"] = "ui"
			if err := atomicWriteFile(path, marshalIndentJSON(d)); err != nil {
				return nil, err
			}
			push("held", did)
		case "approve":
			d["status"] = "approved"
			d["decided_at"] = now
			d["decided_by"] = "ui"
			cd, err := c.campaignDir(camp)
			if err != nil {
				return nil, err
			}
			approvedDir := filepath.Join(cd, "outbox", "approved")
			if err := os.MkdirAll(approvedDir, 0o755); err != nil {
				return nil, err
			}
			dest := filepath.Join(approvedDir, did+".json")
			if err := atomicWriteFile(path, marshalIndentJSON(d)); err != nil {
				return nil, err
			}
			if err := os.Rename(path, dest); err != nil {
				return nil, err
			}
			if err := c.approvalLog(d, "approve", "ui", ""); err != nil {
				return nil, err
			}
			push("approved", map[string]any{"draft_id": did, "path": dest})
		}
	}
	result["processed_lines"] = len(lines)
	if err := os.MkdirAll(filepath.Dir(cursorPath), 0o755); err != nil {
		return nil, err
	}
	if err := atomicWriteFile(cursorPath, strconv.Itoa(len(lines))); err != nil {
		return nil, err
	}
	return result, nil
}

// --- follow-ups --------------------------------------------------------------------

func (c *crmStore) followupsDue(campaignSlug, now string) ([]map[string]any, error) {
	if now == "" {
		now = nowISO()
	}
	cfg := c.getCampaign(campaignSlug)
	if cfg == nil {
		return nil, storageErrf("campaign %q not found", campaignSlug)
	}
	gapByStep := map[int]int{}
	maxStep := 1
	hasSteps := false
	for _, s := range mapsOf(mList(cfg, "sequence")) {
		step := mInt(s, "step", 0)
		gapByStep[step] = mInt(s, "gap_days", 0)
		if !hasSteps || step > maxStep {
			maxStep = step
		}
		hasSteps = true
	}
	type sentState struct {
		Step   int
		SentAt string
	}
	state := map[string]sentState{}
	for _, p := range c.allSentLogs(campaignSlug) {
		for _, r := range readJSONLines(p) {
			if mStr(r, "rfc_message_id") == "" {
				continue
			}
			lid := c.resolve(mStr(r, "lead_id"))
			st := mInt(r, "step", 1)
			sa := mStr(r, "sent_at")
			cur, ok := state[lid]
			if !ok {
				cur = sentState{0, ""}
			}
			if st > cur.Step || (st == cur.Step && sa > cur.SentAt) {
				state[lid] = sentState{st, sa}
			}
		}
	}
	openDrafts, err := c.openDraftSteps(campaignSlug)
	if err != nil {
		return nil, err
	}
	lids := make([]string, 0, len(state))
	for lid := range state {
		lids = append(lids, lid)
	}
	sort.Strings(lids) // deterministic iteration (Python dict order is insertion; consumers treat as set)
	due := []map[string]any{}
	for _, lid := range lids {
		s := state[lid]
		ct := c.getContact(lid)
		if ct == nil || mStr(ct, "sequence_state") == "frozen" {
			continue
		}
		nextStep := s.Step + 1
		if nextStep > maxStep {
			continue
		}
		gap, ok := gapByStep[nextStep]
		if !ok {
			continue
		}
		if openDrafts[leadStep{lid, nextStep}] {
			continue
		}
		if s.SentAt != "" && s.SentAt <= isoDaysAgoFrom(gap, now) {
			due = append(due, map[string]any{"lead_id": lid, "next_step": nextStep,
				"last_step": s.Step, "last_sent_at": s.SentAt})
		}
	}
	return due, nil
}

// --- Today View + kanban ------------------------------------------------------------

func (c *crmStore) todayViewData(now string) map[string]any {
	if now == "" {
		now = nowISO()
	}
	var tasks []map[string]any
	for _, t := range c.latestTasks() {
		if mStr(t, "status") == "open" {
			tasks = append(tasks, t)
		}
	}
	dueTasks := []any{}
	for _, t := range tasks {
		if d := mStr(t, "due_at"); d != "" && d <= now {
			dueTasks = append(dueTasks, t)
		}
	}
	deals, _ := c.a.query("deals", []cond{{Field: "status", Op: "=", Value: "open"}}, nil, -1, 0)
	stageSLA := map[string]any{}
	for _, p := range mapsOf(mList(c.getPipelines(), "pipelines")) {
		for _, s := range mapsOf(mList(p, "stages")) {
			stageSLA[mStr(s, "id")] = s["sla_days"]
		}
	}
	sla := []any{}
	for _, d := range deals {
		hist := mapsOf(mList(d, "stage_history"))
		entered := mStr(d, "created_at")
		if len(hist) > 0 {
			entered = mStr(hist[len(hist)-1], "at")
		}
		sdv := stageSLA[mStr(d, "stage")]
		if !truthy(sdv) {
			continue
		}
		sd := int(asFloat(sdv, 0))
		if entered == "" || entered <= isoDaysAgoFrom(sd, now) {
			since := entered
			if since == "" {
				since = "unknown"
			}
			sla = append(sla, map[string]any{"deal_id": d["id"], "stage": d["stage"],
				"since": since, "sla_days": sdv})
		}
	}
	acts, _ := c.a.readLog("activities", -1, nil)
	var hot []map[string]any
	for _, a := range acts {
		if mStr(a, "type") == "email_reply" {
			hot = append(hot, a)
		}
	}
	if len(hot) > 20 {
		hot = hot[len(hot)-20:]
	}
	hotOut := []any{}
	for _, a := range hot {
		hotOut = append(hotOut, map[string]any{"lead_id": a["contact_id"], "at": a["ts"]})
	}
	return map[string]any{"generated_at": now,
		"tasks_due": dueTasks, "open_tasks": len(tasks),
		"deals_open": len(deals), "sla_breaches": sla,
		"hot_replies":    hotOut,
		"drafts_pending": len(c.listPendingDrafts(""))}
}

func (c *crmStore) renderTodayView(now string) (map[string]any, error) {
	d := c.todayViewData(now)
	gen := mStr(d, "generated_at")
	md := []string{fmt.Sprintf("# Today — %s", strings.Replace(gen[:16], "T", " ", 1)),
		fmt.Sprintf("**%d** drafts awaiting approval  ·  **%d** tasks due  ·  **%d** deals past SLA  ·  **%d** open deals",
			mInt(d, "drafts_pending", 0), len(mList(d, "tasks_due")), len(mList(d, "sla_breaches")), mInt(d, "deals_open", 0)), ""}
	md = append(md, "## Tasks due", "")
	if tasks := mapsOf(mList(d, "tasks_due")); len(tasks) > 0 {
		for _, t := range tasks {
			due := mStr(t, "due_at")
			if len(due) > 16 {
				due = due[:16]
			}
			md = append(md, fmt.Sprintf("- %s  (due %s)", mStr(t, "title"), due))
		}
	} else {
		md = append(md, "*(none)*")
	}
	md = append(md, "", "## Deals past SLA", "")
	if brs := mapsOf(mList(d, "sla_breaches")); len(brs) > 0 {
		for _, b := range brs {
			id := mStr(b, "deal_id")
			if len(id) > 10 {
				id = id[:10]
			}
			since := mStr(b, "since")
			if len(since) > 10 {
				since = since[:10]
			}
			md = append(md, fmt.Sprintf("- deal %s stuck at `%s` since %s (SLA %sd)",
				id, mStr(b, "stage"), since, pyNum(b["sla_days"])))
		}
	} else {
		md = append(md, "*(none)*")
	}
	md = append(md, "", "## Hot replies (respond fast)", "")
	if hrs := mapsOf(mList(d, "hot_replies")); len(hrs) > 0 {
		for _, r := range hrs {
			at := mStr(r, "at")
			if len(at) > 16 {
				at = at[:16]
			}
			md = append(md, fmt.Sprintf("- reply from %s at %s", mStr(r, "lead_id"), at))
		}
	} else {
		md = append(md, "*(none)*")
	}
	return c.renderOperator(md, "today-view", "Today View", now)
}

func (c *crmStore) renderKanban(now string) (map[string]any, error) {
	if now == "" {
		now = nowISO()
	}
	pipelines := mapsOf(mList(c.getPipelines(), "pipelines"))
	deals, _ := c.a.query("deals", []cond{{Field: "status", Op: "=", Value: "open"}}, nil, -1, 0)
	md := []string{fmt.Sprintf("# Pipeline — %s", now[:10]), ""}
	forecast := 0.0
	for _, p := range pipelines {
		for _, st := range mapsOf(mList(p, "stages")) {
			id := mStr(st, "id")
			if id == "won" || id == "lost" {
				continue
			}
			var col []map[string]any
			for _, d := range deals {
				if mStr(d, "stage") == id {
					col = append(col, d)
				}
			}
			md = append(md, fmt.Sprintf("## %s (%d)", id, len(col)))
			for _, d := range col {
				v := asFloat(d["value"], 0)
				prob := asFloat(d["probability"], 0)
				forecast += v * prob
				name := mStr(d, "name")
				if name == "" {
					name = mStr(d, "id")
					if len(name) > 10 {
						name = name[:10]
					}
				}
				md = append(md, fmt.Sprintf("- %s — $%.0f × %s", name, v, pyPercent0(prob)))
			}
			md = append(md, "")
		}
	}
	head := fmt.Sprintf("**Weighted forecast:** $%s  ·  %d open deals\n", commaF0(forecast), len(deals))
	md = append(md[:1], append([]string{head}, md[1:]...)...)
	return c.renderOperator(md, "kanban", "Pipeline Kanban", now)
}

// pyPercent0 mirrors Python f"{x:.0%}" (round-half-even).
func pyPercent0(x float64) string {
	return strconv.FormatFloat(roundHalfEven(x*100, 0), 'f', 0, 64) + "%"
}

func roundHalfEven(x float64, decimals int) float64 {
	shift := 1.0
	for i := 0; i < decimals; i++ {
		shift *= 10
	}
	v := x * shift
	floor := float64(int64(v))
	diff := v - floor
	switch {
	case diff > 0.5:
		floor++
	case diff == 0.5:
		if int64(floor)%2 != 0 {
			floor++
		}
	}
	return floor / shift
}

// commaF0 mirrors Python f"{x:,.0f}".
func commaF0(x float64) string {
	s := strconv.FormatFloat(roundHalfEven(x, 0), 'f', 0, 64)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := strings.Join(parts, ",")
	if neg {
		out = "-" + out
	}
	return out
}

// --- weekly / monthly client reports -------------------------------------------------

var stageLabels = map[string]string{"new_reply": "New replies", "engaged": "In conversation",
	"meeting_booked": "Meeting booked", "proposal_sent": "Proposal sent",
	"won": "Won", "lost": "Closed out"}

func stageLabel(stage string) string {
	if l, ok := stageLabels[stage]; ok {
		return l
	}
	return stage
}

func (c *crmStore) contactDisplay(leadID string) string {
	if leadID == "" {
		return "A prospect"
	}
	ct := c.getContact(c.resolve(leadID))
	if ct == nil {
		return "A prospect"
	}
	nm := strings.TrimSpace(mStr(mMap(ct, "name"), "full"))
	if nm == "" {
		return "A prospect"
	}
	return nm
}

func (c *crmStore) reportWindowData(startISO, endISOExclusive, now string) map[string]any {
	if now == "" {
		now = nowISO()
	}
	inWindow := func(ts string) bool {
		if ts < startISO {
			return false
		}
		return endISOExclusive == "" || ts < endISOExclusive
	}
	acts, _ := c.a.readLog("activities", -1, nil)
	delivered, replies := 0, 0
	for _, a := range acts {
		if !inWindow(mStr(a, "ts")) {
			continue
		}
		switch mStr(a, "type") {
		case "email_sent":
			delivered++
		case "email_reply":
			replies++
		}
	}
	deals, _ := c.a.query("deals", []cond{{Field: "status", Op: "=", Value: "open"}}, nil, -1, 0)
	forecast := 0.0
	byStage := map[string]any{}
	for _, d := range deals {
		v := asFloat(d["value"], 0)
		prob := asFloat(d["probability"], 0)
		forecast += v * prob
		st := mStr(d, "stage")
		agg, ok := byStage[st].(map[string]any)
		if !ok {
			agg = map[string]any{"count": 0, "value": 0.0}
			byStage[st] = agg
		}
		agg["count"] = agg["count"].(int) + 1
		agg["value"] = agg["value"].(float64) + v
	}
	allDeals, _ := c.a.query("deals", nil, nil, -1, 0)
	movements := []map[string]any{}
	for _, d := range allDeals {
		cids := mList(d, "contact_ids")
		first := ""
		if len(cids) > 0 {
			first, _ = cids[0].(string)
		}
		name := c.contactDisplay(first)
		status := mStr(d, "status")
		if status == "" {
			status = "open"
		}
		for _, h := range mapsOf(mList(d, "stage_history")) {
			if inWindow(mStr(h, "at")) {
				movements = append(movements, map[string]any{"name": name, "stage": mStr(h, "stage"),
					"at": mStr(h, "at"), "value": asFloat(d["value"], 0), "status": status})
			}
		}
	}
	sort.SliceStable(movements, func(i, j int) bool {
		return mStr(movements[i], "at") < mStr(movements[j], "at")
	})
	meetings := 0
	newConversations := 0
	var won []any
	movementsAny := make([]any, len(movements))
	for i, m := range movements {
		movementsAny[i] = m
		switch mStr(m, "stage") {
		case "meeting_booked":
			meetings++
		case "new_reply":
			newConversations++
		case "won":
			won = append(won, m)
		}
	}
	nextSteps := []any{}
	for _, d := range deals {
		st := mStr(d, "stage")
		if st == "meeting_booked" || st == "proposal_sent" {
			cids := mList(d, "contact_ids")
			first := ""
			if len(cids) > 0 {
				first, _ = cids[0].(string)
			}
			nextSteps = append(nextSteps, map[string]any{"name": c.contactDisplay(first), "stage": st})
		}
	}
	var replyRate any
	if delivered > 0 {
		rr := float64(replies) / float64(delivered)
		if rr > 1.0 {
			rr = 1.0
		}
		replyRate = rr
	}
	return map[string]any{"generated_at": now,
		"delivered": delivered, "replies": replies, "reply_rate": replyRate,
		"new_conversations": newConversations, "meetings": meetings,
		"open_deals": len(deals), "forecast": forecast, "by_stage": byStage,
		"movements": movementsAny, "won": orEmptyList(won), "next_steps": nextSteps}
}

func (c *crmStore) weeklyReportData(now string, days int) map[string]any {
	if now == "" {
		now = nowISO()
	}
	cutoff := isoDaysAgoFrom(days, now)[:10] + "T00:00:00Z"
	d := c.reportWindowData(cutoff, "", now)
	d["period_start"] = cutoff[:10]
	d["period_end"] = now[:10]
	d["days"] = days
	return d
}

func (c *crmStore) monthlyReportData(month, now string) map[string]any {
	if now == "" {
		now = nowISO()
	}
	if month == "" {
		month = now[:7]
	}
	start := month + "-01T00:00:00Z"
	current := month == now[:7]
	nxt := monthEndExclusive(month)
	end := nxt
	if current {
		end = ""
	}
	d := c.reportWindowData(start, end, now)
	d["period_start"] = start[:10]
	if current {
		d["period_end"] = now[:10]
	} else {
		d["period_end"] = nxt[:10]
	}
	d["month"] = month
	return d
}

func (c *crmStore) clientReportMD(d map[string]any, cname, titleKind string) []string {
	period := "month"
	if titleKind == "Weekly" {
		period = "week"
	}
	rr := "—"
	if v, ok := toFloat(d["reply_rate"]); ok && d["reply_rate"] != nil {
		rr = pyPercent0(v)
	}
	md := []string{fmt.Sprintf("# %s — %s Outreach Report", cname, titleKind),
		fmt.Sprintf("### %s to %s", mStr(d, "period_start"), mStr(d, "period_end")), "",
		fmt.Sprintf("**This %s:** %d emails delivered · %d replies · %d new conversations · $%s in active pipeline",
			period, mInt(d, "delivered", 0), mInt(d, "replies", 0), mInt(d, "new_conversations", 0),
			commaF0(asFloat(d["forecast"], 0))), "",
		"## Snapshot", "",
		fmt.Sprintf("- Emails delivered: **%d**", mInt(d, "delivered", 0)),
		fmt.Sprintf("- Replies received: **%d** (%s)", mInt(d, "replies", 0), rr),
		fmt.Sprintf("- New conversations started: **%d**", mInt(d, "new_conversations", 0)),
		fmt.Sprintf("- Meetings booked: **%d**", mInt(d, "meetings", 0)),
		fmt.Sprintf("- Active opportunities: **%d**", mInt(d, "open_deals", 0)), "",
		"## Pipeline", "",
		fmt.Sprintf("**Weighted pipeline value: $%s** across %d opportunities",
			commaF0(asFloat(d["forecast"], 0)), mInt(d, "open_deals", 0)), "",
		"| Stage | Opportunities | Value |", "|---|---:|---:|"}
	byStage := mMap(d, "by_stage")
	for _, st := range []string{"new_reply", "engaged", "meeting_booked", "proposal_sent"} {
		if agg, ok := byStage[st].(map[string]any); ok {
			md = append(md, fmt.Sprintf("| %s | %s | $%s |", stageLabels[st],
				pyNum(agg["count"]), commaF0(asFloat(agg["value"], 0))))
		}
	}
	md = append(md, "", fmt.Sprintf("## What moved this %s", period), "")
	var moved []string
	for _, m := range mapsOf(mList(d, "movements")) {
		label := stageLabel(mStr(m, "stage"))
		switch mStr(m, "stage") {
		case "won":
			moved = append(moved, fmt.Sprintf("- **Won** — %s ($%s)", mStr(m, "name"), commaF0(asFloat(m["value"], 0))))
		case "lost":
			moved = append(moved, fmt.Sprintf("- Closed out — %s", mStr(m, "name")))
		default:
			moved = append(moved, fmt.Sprintf("- %s → %s", mStr(m, "name"), label))
		}
	}
	if len(moved) > 0 {
		md = append(md, moved...)
	} else {
		md = append(md, fmt.Sprintf("Conversations are progressing; no stage changes to report this %s.", period))
	}
	md = append(md, "", "## What's next", "")
	nexts := mapsOf(mList(d, "next_steps"))
	if len(nexts) > 0 {
		for _, n := range nexts {
			md = append(md, fmt.Sprintf("- %s — %s in progress", mStr(n, "name"),
				strings.ToLower(stageLabel(mStr(n, "stage")))))
		}
	} else {
		md = append(md, "We continue outreach and follow-ups on the active list.")
	}
	md = append(md, "", "---", fmt.Sprintf("Prepared by your outreach team · %s", mStr(d, "period_end")))
	return md
}

func (c *crmStore) reportClientName(clientName string) string {
	cname := strings.TrimSpace(clientName)
	if cname == "" {
		cname = pyTitle(strings.ReplaceAll(strings.ReplaceAll(c.clientSlug(), "-", " "), "_", " "))
	}
	return cname
}

func (c *crmStore) renderWeeklyReport(now, clientName string, days int) (map[string]any, error) {
	d := c.weeklyReportData(now, days)
	cname := c.reportClientName(clientName)
	md := c.clientReportMD(d, cname, "Weekly")
	return c.renderClientFacing(md, "weekly-client-report", cname+" — Weekly Report",
		cname, "Weekly Client Report", now)
}

func (c *crmStore) renderMonthlyReport(now, clientName, month string) (map[string]any, error) {
	d := c.monthlyReportData(month, now)
	cname := c.reportClientName(clientName)
	md := c.clientReportMD(d, cname, "Monthly")
	return c.renderClientFacing(md, "monthly-client-report", cname+" — Monthly Report",
		cname, "Monthly Client Report", now)
}

func (c *crmStore) renderOperator(mdLines []string, slugSuffix, title, now string) (map[string]any, error) {
	if now == "" {
		now = nowISO()
	}
	outDir := filepath.Join(c.clientDir, "outputs", todayStr(now))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	mdPath := filepath.Join(outDir, c.clientSlug()+"-"+slugSuffix+".md")
	htmlPath := filepath.Join(outDir, c.clientSlug()+"-"+slugSuffix+".html")
	if err := atomicWriteFile(mdPath, strings.Join(mdLines, "\n")); err != nil {
		return nil, err
	}
	res := renderReportFile(rendererRequest{Input: mdPath, OutputHTML: htmlPath,
		Title: title, ReportDate: now[:10]})
	out := map[string]any{"md": mdPath, "html": nil, "html_rendered": res.RC == 0}
	if res.RC == 0 {
		out["html"] = htmlPath
	}
	return out, nil
}

func (c *crmStore) renderClientFacing(mdLines []string, slugSuffix, title, clientName, reportKind, now string) (map[string]any, error) {
	if now == "" {
		now = nowISO()
	}
	outDir := filepath.Join(c.clientDir, "outputs", todayStr(now))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	mdPath := filepath.Join(outDir, c.clientSlug()+"-"+slugSuffix+".md")
	htmlPath := filepath.Join(outDir, c.clientSlug()+"-"+slugSuffix+".html")
	if err := atomicWriteFile(mdPath, strings.Join(mdLines, "\n")); err != nil {
		return nil, err
	}
	res := renderReportFile(rendererRequest{Input: mdPath, OutputHTML: htmlPath,
		Title: title, ClientName: clientName, ReportKind: reportKind, ReportDate: now[:10],
		ClientFacing: true, FailOnScrub: true})
	if res.RC == 3 {
		return map[string]any{"md": mdPath, "html": nil, "html_rendered": false,
			"blocked": true, "blind_terms": orEmptyList(res.BlindTerms)}, nil
	}
	rendered := res.RC == 0 || res.RC == 2
	out := map[string]any{"md": mdPath, "html": nil, "html_rendered": rendered, "blocked": false}
	if rendered {
		out["html"] = htmlPath
	}
	return out, nil
}
