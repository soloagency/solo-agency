package main

import "testing"

// TestZillowURLShapeClassification locks the Zillow rules added for the collector's
// zillow.agents.list / zillow.profile.enrich capabilities: /profile/<screenName> is a
// PERSON/TEAM anchor (identities.socials.zillow, and accepted by enrich write's
// channels_found.profiles validator), the agent DIRECTORY is a query page and never an
// identity (like a search engine), and listing pages are content evidence.
func TestZillowURLShapeClassification(t *testing.T) {
	cases := []struct{ url, wantKind, wantPlatform, wantSub string }{
		{"https://www.zillow.com/profile/Pardee%20Properties", "profile", "zillow", ""},
		{"https://www.zillow.com/profile/tykunkle", "profile", "zillow", ""},
		{"https://www.zillow.com/profile/tykunkle/", "profile", "zillow", ""},
		{"zillow.com/profile/agentjasonkim", "profile", "zillow", ""},
		{"https://www.zillow.com/profile/tykunkle#reviews", "profile", "zillow", ""},
		// the directory / keyword search is a QUERY, never an identity
		{"https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=kim", "", "", ""},
		{"https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?page=2", "", "", ""},
		{"https://www.zillow.com/professionals/real-estate-agent-reviews/ca/", "", "", ""},
		// content on the host is evidence (a hook), not a person
		{"https://www.zillow.com/homedetails/4834-McConnell-Ave-Los-Angeles-CA-90066/20441325_zpid/", "seed", "zillow", "post"},
		{"https://www.zillow.com/los-angeles-ca/", "seed", "zillow", "post"},
	}
	for _, c := range cases {
		kind, platform, sub := classifyLeadURLFull(c.url)
		if kind != c.wantKind || platform != c.wantPlatform || sub != c.wantSub {
			t.Errorf("%s => (%s,%s,%s), want (%s,%s,%s)", c.url, kind, platform, sub, c.wantKind, c.wantPlatform, c.wantSub)
		}
	}
	// the two-value wrapper agrees, and a profile stays a profile through it
	if k, p := classifyLeadURL("https://www.zillow.com/profile/tykunkle"); k != "profile" || p != "zillow" {
		t.Errorf("classifyLeadURL zillow profile => (%s,%s)", k, p)
	}
	// unrelated hosts that merely contain the word are not Zillow
	if k, p, _ := classifyLeadURLFull("https://notzillow.com/profile/x"); p == "zillow" || k == "profile" {
		t.Errorf("notzillow.com must not classify as a zillow profile: (%s,%s)", k, p)
	}
}
