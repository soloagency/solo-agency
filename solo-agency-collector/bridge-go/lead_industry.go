package main

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
)

// leadIndustries is the closed set a contact's enrichment.identity.industry may hold. It is parsed
// once from the embedded lead_industries.json, which is also the file the enrichment playbook tells
// the agent to read — one source of bytes, so the gate and the instruction cannot drift apart.
var (
	leadIndustryOnce sync.Once
	leadIndustrySet  map[string]bool
	leadIndustryList []string
)

func loadLeadIndustries() {
	leadIndustryOnce.Do(func() {
		leadIndustrySet = map[string]bool{}
		var doc struct {
			Industries []struct {
				Industry string `json:"industry"`
			} `json:"industries"`
		}
		if err := json.Unmarshal(embeddedLeadIndustries, &doc); err != nil {
			return
		}
		for _, e := range doc.Industries {
			v := strings.TrimSpace(e.Industry)
			if v == "" {
				continue
			}
			leadIndustrySet[v] = true
			leadIndustryList = append(leadIndustryList, v)
		}
		sort.Strings(leadIndustryList)
	})
}

// validLeadIndustry is deliberately CASE- AND PUNCTUATION-SENSITIVE. "real estate" is not
// "Real Estate", and "Tax, Accounting and Bookkeeping" is not "Tax, Accounting, Bookkeeping".
// Normalising here would defeat the point: downstream code matches these strings exactly, and a
// vocabulary that silently accepts near-misses is a vocabulary with more than 42 members.
func validLeadIndustry(v string) bool {
	loadLeadIndustries()
	return leadIndustrySet[v]
}

// sortedLeadIndustries backs the error message, matching the goalTypes/channelStrategies idiom:
// telling the caller what IS allowed is the difference between a rejection they can act on and one
// they have to go read source code for.
func sortedLeadIndustries() []string {
	loadLeadIndustries()
	return leadIndustryList
}

// jsonUnmarshalLeadIndustries exposes the embedded bytes for tests that assert on the dictionary's
// SHAPE rather than its membership, so those tests read the shipped file instead of a copy.
func jsonUnmarshalLeadIndustries(v any) error { return json.Unmarshal(embeddedLeadIndustries, v) }
