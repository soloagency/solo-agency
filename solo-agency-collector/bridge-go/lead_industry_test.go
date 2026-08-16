package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// The dictionary is the contract. These check the file itself, not a copy of it in the test — a
// test that restates the 42 values would drift from the source exactly the way a second playbook
// copy would, and would then pass while the real vocabulary was wrong.
func TestLeadIndustryDictionaryLoads(t *testing.T) {
	got := sortedLeadIndustries()
	if len(got) != 43 {
		t.Fatalf("expected 43 industries in lead_industries.json, got %d", len(got))
	}
	seen := map[string]bool{}
	for _, v := range got {
		if seen[v] {
			t.Errorf("duplicate industry %q", v)
		}
		seen[v] = true
	}
}

// Case and punctuation are load-bearing. Downstream code matches these strings exactly, so a
// vocabulary that accepts near-misses has more than 42 members and nobody can tell which.
func TestLeadIndustryIsExactMatch(t *testing.T) {
	for _, ok := range []string{"Real Estate", "P&C Insurance", "L&H Insurance"} {
		if !validLeadIndustry(ok) {
			t.Errorf("canonical value %q must be accepted", ok)
		}
	}
	// The merged value is GONE. Leaving it valid would let old records and new ones disagree
	// while both looked correct, which is worse than a rejection anybody can see.
	if validLeadIndustry("Insurance") {
		t.Error(`"Insurance" was split into "P&C Insurance" and "L&H Insurance" and must no longer validate`)
	}
	for _, bad := range []string{
		"real estate",                     // lowercased
		"REAL ESTATE",                     // shouted
		"Real  Estate",                    // doubled space
		" Real Estate",                    // padded
		"Banking and Financial",           // 'and' for '&'
		"Tax, Accounting and Bookkeeping", // rewritten punctuation
		"Realtor",                         // an alias, not the value
		"Mortgage",                        // a 43rd value invented by an agent
		"",
	} {
		if validLeadIndustry(bad) {
			t.Errorf("near-miss %q must NOT be accepted", bad)
		}
	}
}

// Every entry must carry exclusions: sixteen pairs in this list overlap (Real Estate vs
// Loan & Mortgage, Immigration vs Work Abroad & Immigration, Logistics vs Transportation, ...),
// and without a tie-break rule the same lead classifies differently on two runs.
// P&C and L&H are the sharpest boundary in the file: an agent who sells both is exactly the lead
// this split exists to disambiguate, so each must name the other as an exclusion.
func TestInsuranceSplitIsMutuallyExclusive(t *testing.T) {
	var doc struct {
		Industries []struct {
			Industry   string   `json:"industry"`
			Exclusions []string `json:"exclusions"`
		} `json:"industries"`
	}
	if err := jsonUnmarshalLeadIndustries(&doc); err != nil {
		t.Fatal(err)
	}
	ex := map[string]string{}
	for _, e := range doc.Industries {
		ex[e.Industry] = strings.Join(e.Exclusions, " ")
	}
	if !strings.Contains(ex["P&C Insurance"], "L&H Insurance") {
		t.Error("P&C Insurance must exclude L&H Insurance")
	}
	if !strings.Contains(ex["L&H Insurance"], "P&C Insurance") {
		t.Error("L&H Insurance must exclude P&C Insurance")
	}
	// Nothing may still point at the merged value; a dangling "-> Insurance" is an exclusion that
	// no longer resolves, which is an ambiguity dressed as a rule.
	for ind, e := range ex {
		for _, line := range strings.Split(e, " -> ") {
			if strings.TrimSpace(line) == "Insurance" {
				t.Errorf("%q still points at the removed value \"Insurance\"", ind)
			}
		}
	}
}

func TestEveryIndustryHasTieBreakRules(t *testing.T) {
	var doc struct {
		Industries []struct {
			Industry        string   `json:"industry"`
			PositiveSignals []string `json:"positive_signals"`
			Exclusions      []string `json:"exclusions"`
		} `json:"industries"`
	}
	if err := jsonUnmarshalLeadIndustries(&doc); err != nil {
		t.Fatalf("lead_industries.json does not parse: %v", err)
	}
	for _, e := range doc.Industries {
		if len(e.PositiveSignals) == 0 {
			t.Errorf("%q has no positive_signals", e.Industry)
		}
		if len(e.Exclusions) == 0 {
			t.Errorf("%q has no exclusions — an entry with no boundary is an entry that collides", e.Industry)
		}
	}
}

// The gate lives in enrichWrite, so it is exercised through the CLI the playbook actually calls.
func TestEnrichIndustryGate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	if r := runGoStep(t, xstep{"2026-08-15T08:00:00Z", []string{"--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"}}); r.Code != 0 {
		t.Fatal(r.Stderr)
	}
	if r := runGoStep(t, xstep{"2026-08-15T08:01:00Z", []string{"--client-dir", ws, "contact", "add",
		"--json", `{"id": "c_ind", "name": {"full": "Grant Alpine"}, "identities": {"emails": [{"address": "grant@alpfinancing.com", "is_primary": true}]}}`}}); r.Code != 0 {
		t.Fatal(r.Stderr)
	}
	enrich := func(ts, dossier string) map[string]any {
		r := runGoStep(t, xstep{ts, []string{"--client-dir", ws, "enrich", "write", "--contact", "c_ind", "--json", dossier}})
		if r.Code != 0 {
			t.Fatal(r.Stderr)
		}
		return parseOut(t, r)
	}
	stored := func() string {
		r := runGoStep(t, xstep{"2026-08-15T09:00:00Z", []string{"--client-dir", ws, "contact", "get", "--id", "c_ind"}})
		if r.Code != 0 {
			t.Fatal(r.Stderr)
		}
		return mStr(mMap(mMap(parseOut(t, r), "enrichment"), "identity"), "industry")
	}
	problems := func(res map[string]any) string {
		var sb strings.Builder
		for _, p := range mList(res, "problems") {
			sb.WriteString(fmt.Sprint(p) + " | ")
		}
		return sb.String()
	}

	// A canonical value persists.
	enrich("2026-08-15T08:05:00Z", `{"identity": {"still_active": "confirmed", "industry": "Loan & Mortgage",
		"industry_confidence": 0.9, "industry_reason": "NMLS #2509012 names the licensed trade",
		"industry_signals": ["NMLS #2509012"]}}`)
	if got := stored(); got != "Loan & Mortgage" {
		t.Fatalf("canonical industry did not persist, got %q", got)
	}

	// A near-miss is REJECTED and REPORTED. Accepting it would grow the vocabulary past 42 without
	// anyone deciding to, and downstream code matches these strings exactly.
	res := enrich("2026-08-15T08:06:00Z", `{"identity": {"still_active": "confirmed", "industry": "mortgage"}}`)
	if !strings.Contains(problems(res), "not in the lead industry dictionary") {
		t.Fatalf("a value outside the dictionary must be reported, problems=%q", problems(res))
	}
	// ...and the good value it tried to overwrite SURVIVES. Dropping the key lets the merge fall
	// back to what was already stored; without that, one typo silently erases a correct label.
	if got := stored(); got != "Loan & Mortgage" {
		t.Fatalf("a rejected write destroyed the stored value, got %q", got)
	}

	// Empty is legitimate: the playbook requires an empty industry over a guessed one, because a
	// wrong label mis-targets every later message in a way an empty field does not.
	res = enrich("2026-08-15T08:07:00Z", `{"identity": {"still_active": "confirmed", "industry": ""}}`)
	if strings.Contains(problems(res), "not in the lead industry dictionary") {
		t.Fatalf("an empty industry must not be reported as invalid, problems=%q", problems(res))
	}
}
