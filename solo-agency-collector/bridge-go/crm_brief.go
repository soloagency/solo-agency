package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// crm_brief.go — `tool crm-store draft brief`: hand the writer everything the
// system already knows, and prove it was handed over.
//
// The refactor this serves: the operator answers ONE campaign question (the
// goal, in their own words) and the rest of the persuasion inputs come from
// what setup already collected. That only works if the data actually REACHES
// the writer — playbook 06 told the agent to "compose from the client profile"
// for weeks while the drafting path never opened the profile once, and 53 live
// drafts came out 55% identical. Hope is not a delivery mechanism. This is:
// one command assembles the full write-time context, writes it to disk as the
// auditable record of what the writer saw, and `draft write` refuses to store
// a draft for a described campaign unless that record exists.

// profileBriefSections are the client-profile sections a writer needs. The
// profile is the setup-time interview; these are the persuasion-bearing parts
// (who the client is, who they serve, what hurts those people, why the client
// is credible, how they sound). Operational sections (sendboxes, automation,
// suppression files) stay out — the writer has no business with them.
var profileBriefSections = []string{
	"business_description", "offer", "icp", "pain_points", "value_prop",
	"proof_points", "brand_voice", "language", "compliance_notes",
}

var profileBackupRe = regexp.MustCompile(`_\d{8}_\d{4}\.md$`)

// clientProfilePath finds the canonical client_profile_*.md (not the
// timestamped backups the agents leave beside it).
func (c *crmStore) clientProfilePath() string {
	matches, _ := filepath.Glob(filepath.Join(c.clientDir, "client_profile_*.md"))
	sort.Strings(matches)
	for _, m := range matches {
		if !profileBackupRe.MatchString(m) {
			return m
		}
	}
	return ""
}

// profileSections splits a client-profile markdown file into its "## name"
// sections and returns the whitelisted ones verbatim. Raw text on purpose: the
// consumer is the writing agent, and the profile's value:/status:/rationale:
// convention is more faithfully carried as-is than re-parsed into a lossy map.
func profileSections(raw string) map[string]string {
	want := map[string]bool{}
	for _, s := range profileBriefSections {
		want[s] = true
	}
	out := map[string]string{}
	var name string
	var buf []string
	flush := func() {
		if name != "" && want[name] {
			out[name] = strings.TrimSpace(strings.Join(buf, "\n"))
		}
	}
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			name = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			buf = nil
			continue
		}
		buf = append(buf, line)
	}
	flush()
	return out
}

// priorDraftsForLead walks every outbox status dir and returns this lead's
// drafts (any status, any day): the writer must know what earlier touches said,
// and rotation must know which bank messages this lead has already been taught.
func (c *crmStore) priorDraftsForLead(campaignSlug, leadID string) []map[string]any {
	var out []map[string]any
	dir, err := c.campaignDir(campaignSlug)
	if err != nil {
		return nil
	}
	root := filepath.Join(dir, "outbox")
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasPrefix(info.Name(), "draft_") || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}
		d, rerr := readJSONFile(path)
		if rerr != nil || mStr(d, "lead_id") != leadID {
			return nil
		}
		out = append(out, d)
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return mStr(out[i], "created_at") < mStr(out[j], "created_at") })
	return out
}

// bankRotation splits the campaign's message bank into what this lead has
// already been taught and what is still fresh. A follow-up that re-teaches the
// same message is the sequence coming out "the same color" — the exact failure
// the bank exists to prevent.
func bankRotation(bank []any, prior []map[string]any) (used, fresh []string) {
	usedSet := map[string]bool{}
	for _, d := range prior {
		if mStr(d, "status") == "rejected" {
			continue // a rejected draft never reached the lead; its messages are unspent
		}
		for _, m := range mList(d, "bank_messages_used") {
			usedSet[strings.TrimSpace(strings.ToLower(sprint(m)))] = true
		}
	}
	for _, e := range mapsOf(bank) {
		msg := mStr(e, "msg")
		if msg == "" {
			continue
		}
		if usedSet[strings.TrimSpace(strings.ToLower(msg))] {
			used = append(used, msg)
		} else {
			fresh = append(fresh, msg)
		}
	}
	return used, fresh
}

func sprint(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return strings.TrimSpace(string(marshalLineJSON(v)))
}

func (c *crmStore) briefPath(campaignSlug, leadID string) (string, error) {
	dir, err := c.campaignDir(campaignSlug)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "queue", "briefs", leadID+".json"), nil
}

// draftBrief assembles the write-time context for one lead and persists it.
// The stored file is both the delivery and the proof of delivery: what the
// writer had in hand is exactly what an auditor later reads.
func (c *crmStore) draftBrief(contactID, campaignSlug, now string) (map[string]any, error) {
	if now == "" {
		now = nowISO()
	}
	leadID := c.resolve(contactID)
	ct := c.getContact(leadID)
	if ct == nil {
		return nil, storageErrf("contact_not_found: %q", contactID)
	}
	cfg := c.getCampaign(campaignSlug)
	if cfg == nil {
		return nil, storageErrf("campaign %q not found", campaignSlug)
	}
	goal := mMap(cfg, "goal")

	client := map[string]any{}
	if p := c.clientProfilePath(); p != "" {
		if raw, err := os.ReadFile(p); err == nil {
			for k, v := range profileSections(string(raw)) {
				client[k] = v
			}
		}
	}

	prior := c.priorDraftsForLead(campaignSlug, leadID)
	bank := mList(goal, "message_bank")
	used, fresh := bankRotation(bank, prior)

	history := make([]map[string]any, 0, len(prior))
	for _, d := range prior {
		hooks := []string{}
		for _, h := range mapsOf(mList(d, "hooks_used")) {
			hooks = append(hooks, mStr(h, "evidence_url"))
		}
		history = append(history, map[string]any{
			"step": d["step"], "status": mStr(d, "status"), "subject": mStr(d, "subject"),
			"created_at": mStr(d, "created_at"), "bank_messages_used": mList(d, "bank_messages_used"),
			"hook_urls": hooks,
		})
	}

	en := mMap(ct, "enrichment")
	ids := mMap(ct, "identities")
	brief := map[string]any{
		"schema_version": 1, "issued_at": now,
		"campaign": campaignSlug, "lead_id": leadID,
		"client": client,
		// description + profile ARE the goal; the old objective/offer/value_prop
		// campaign fields were copies of profile sections the brief already
		// carries, and two sources for one fact is how contradictions ship.
		"campaign_goal": map[string]any{
			"description":   mStr(goal, "description"),
			"goal_type":     mStr(goal, "goal_type"),
			"cta":           mMap(goal, "cta"),
			"companion_doc": mMap(goal, "companion_doc"),
			"message_bank":  bank,
		},
		"signature": c.senderSignature(),
		"bank_rotation": map[string]any{
			"already_taught_this_lead": orEmptyList(anySlice(used)),
			"fresh_for_this_lead":      orEmptyList(anySlice(fresh)),
		},
		"lead": map[string]any{
			"id": leadID, "name": mMap(ct, "name"),
			"emails": mList(ids, "emails"), "socials": mMap(ids, "socials"),
			"website": ids["website"], "seeds": mList(ids, "seeds"),
			"custom_fields":   mMap(ct, "custom_fields"),
			"hooks":           mList(en, "hooks"),
			"writing_brief":   mMap(en, "writing_brief"),
			"confidence_band": mStr(en, "confidence_band"),
		},
		"history": history,
	}

	path, err := c.briefPath(campaignSlug, leadID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := atomicWriteFile(path, marshalIndentJSON(brief)); err != nil {
		return nil, err
	}
	return brief, nil
}

var profileFieldRe = regexp.MustCompile(`(?m)^(from_name|from_title|signature_block):\s*(.+)$`)

// senderSignature pulls who signs from the profile's sending_identity — the
// writer must end the email as this person, and the missing_signature gate
// verifies it did.
func (c *crmStore) senderSignature() map[string]any {
	out := map[string]any{}
	p := c.clientProfilePath()
	if p == "" {
		return out
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return out
	}
	for _, m := range profileFieldRe.FindAllStringSubmatch(string(raw), -1) {
		out[m[1]] = strings.TrimSpace(m[2])
	}
	return out
}

func anySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
