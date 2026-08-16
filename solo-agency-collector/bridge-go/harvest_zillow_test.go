package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestZillowURLComposition(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=", "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/"},
		{"https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=kim&page=3", "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/"},
		{"zillow.com/professionals/real-estate-agent-reviews/venice-ca", "https://www.zillow.com/professionals/real-estate-agent-reviews/venice-ca/"},
	}
	for _, c := range cases {
		got, err := zillowDirectoryURL(c.in)
		if err != nil || got != c.want {
			t.Errorf("zillowDirectoryURL(%q) = (%q, %v), want %q", c.in, got, err, c.want)
		}
	}
	if _, err := zillowDirectoryURL("https://www.facebook.com/groups/x"); err == nil {
		t.Fatal("non-zillow url must be refused")
	}
	if _, err := zillowDirectoryURL("https://www.zillow.com/homes/los-angeles-ca/"); err == nil {
		t.Fatal("non-professionals path must be refused")
	}
	// The composed page url is exactly the operator's shape, from any pasted form.
	loc, _ := zillowDirectoryURL("https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=")
	if got := zillowPageURL(loc, "kim", 3); got != "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=kim&page=3" {
		t.Fatalf("page url wrong: %s", got)
	}
}

func TestZillowCursorWalk(t *testing.T) {
	zc := zillowConfig{
		Locations: []string{"https://www.zillow.com/professionals/real-estate-agent-reviews/la-ca/", "https://www.zillow.com/professionals/real-estate-agent-reviews/oc-ca/"},
		Keywords:  []string{"kim", "nguyen"},
	}
	c := &zillowCursor{Page: 1, QueriesDone: map[string]int{}}
	loc, kw, u := zc.currentQuery(c)
	if loc != zc.Locations[0] || kw != "kim" || u == "" {
		t.Fatalf("first query wrong: %s %s %s", loc, kw, u)
	}
	// page 1 has more -> page 2
	zc.advanceAfterLeg(c, zillowEndOfQuery(zillowLegFacts{Status: "ok", HasMore: true, HasMoreKnown: true, Page: 1, PageCount: 25}, 15, c.Page))
	if c.Page != 2 || c.KwIdx != 0 {
		t.Fatalf("should advance page: %+v", c)
	}
	// missing has_more (degraded) does NOT end the query
	if zillowEndOfQuery(zillowLegFacts{Status: "ok", Page: 2}, 15, c.Page) {
		t.Fatal("unknown has_more must not end the query")
	}
	// degraded read with 0 cards is NOT an end (retried), only a real empty page is
	if zillowEndOfQuery(zillowLegFacts{Status: "no_next_data", Page: 2}, 0, c.Page) {
		t.Fatal("no_next_data + 0 cards must not end the query")
	}
	// page ceiling on the CURSOR page
	if !zillowEndOfQuery(zillowLegFacts{Status: "ok", HasMore: true, HasMoreKnown: true, Page: 1}, 15, zillowMaxPagesPerQuery) {
		t.Fatal("cursor page at the 25-page ceiling must end the query")
	}
	// empty page ends the query -> next keyword, page 1
	zc.advanceAfterLeg(c, zillowEndOfQuery(zillowLegFacts{Status: "empty", Page: 2}, 0, c.Page))
	if c.KwIdx != 1 || c.Page != 1 || c.LocIdx != 0 {
		t.Fatalf("empty page must move to next keyword at page 1: %+v", c)
	}
	// page >= page_count ends the query -> next LOCATION (keywords wrap)
	zc.advanceAfterLeg(c, zillowEndOfQuery(zillowLegFacts{Status: "ok", HasMore: true, HasMoreKnown: true, Page: 25, PageCount: 25}, 15, c.Page))
	if c.LocIdx != 1 || c.KwIdx != 0 || c.Page != 1 {
		t.Fatalf("last page must move to next location: %+v", c)
	}
	// finish location 2 both keywords -> exhausted
	zc.advanceAfterLeg(c, true)
	zc.advanceAfterLeg(c, true)
	if !c.Exhausted {
		t.Fatalf("walk must be exhausted: %+v", c)
	}
	if _, _, u := zc.currentQuery(c); u != "" {
		t.Fatal("exhausted cursor must yield no query")
	}
}

func TestZillowContactMappingAndSkipRule(t *testing.T) {
	rec := map[string]any{
		"name": "Tyler Kunkle", "category": "Real estate agent", "industry": "Real Estate",
		"profile_url": "https://www.zillow.com/profile/tykunkle",
		"emails": []any{"tyler@example.com"}, "phones": []any{"(310) 555-7333"},
		"website": "https://example.com/team/tyler", "location": []any{"Venice, CA 90291"},
		"zillow": map[string]any{"brokerage": "Example Properties", "encoded_zuid": "X1-ZU123",
			"socials": map[string]any{"instagram": "https://www.instagram.com/tykunkle/"}},
	}
	f, ok := zillowContactFields(rec, "https://www.zillow.com/profile/tykunkle", "z-oc", "https://www.zillow.com/professionals/x/?name=kim&page=1")
	if !ok {
		t.Fatal("email+phone record must map")
	}
	ident := mMap(f, "identities")
	if len(mList(ident, "emails")) != 1 || len(mList(ident, "phones")) != 1 || mStr(mMap(ident, "socials"), "zillow") == "" || mStr(mMap(ident, "socials"), "instagram") == "" {
		t.Fatalf("identities wrong: %v", ident)
	}
	// Round-trip through the REAL store: identity items must be {address}/{number}
	// maps or the store silently drops them (the review's critical finding).
	rtRoot := t.TempDir()
	rtStore := newCrmStore(filepath.Join(rtRoot, "clients", "a", "b", "outreach"))
	_ = rtStore.initTree()
	id1, outcome, err := rtStore.addContact(f)
	if err != nil || outcome != "created" {
		t.Fatalf("addContact: %v %s", err, outcome)
	}
	ct := rtStore.getContact(id1)
	em := mapsOf(mList(mMap(ct, "identities"), "emails"))
	ph := mapsOf(mList(mMap(ct, "identities"), "phones"))
	if len(em) != 1 || mStr(em[0], "address") != "tyler@example.com" || len(ph) != 1 || mStr(ph[0], "number") == "" {
		t.Fatalf("stored identities lost email/phone: %v", mMap(ct, "identities"))
	}
	// Same email under a different zillow url must MATCH, not duplicate.
	rec2 := map[string]any{"name": "Tyler K", "profile_url": "https://www.zillow.com/profile/tk2", "emails": []any{"tyler@example.com"}}
	f2, _ := zillowContactFields(rec2, "https://www.zillow.com/profile/tk2", "z-oc", "s")
	id2, outcome2, _ := rtStore.addContact(f2)
	if outcome2 != "matched" || id2 != id1 {
		t.Fatalf("same email must match the existing contact: %s %s vs %s", outcome2, id2, id1)
	}
	cf := mMap(f, "custom_fields")
	if mStr(cf, "industry") != "Real Estate" || mStr(cf, "brokerage") != "Example Properties" || mStr(cf, "source") != "zillow_harvest" {
		t.Fatalf("custom fields wrong: %v", cf)
	}
	// phone only -> still mapped (operator rule b)
	recPhone := map[string]any{"name": "A", "phones": []any{"111"}, "profile_url": "https://www.zillow.com/profile/a"}
	if _, ok := zillowContactFields(recPhone, "", "z", ""); !ok {
		t.Fatal("phone-only record must map")
	}
	// neither -> skipped
	rec3 := map[string]any{"name": "B", "profile_url": "https://www.zillow.com/profile/b"}
	if _, ok := zillowContactFields(rec3, "", "z", ""); ok {
		t.Fatal("no email/phone must be skipped")
	}
	// team members become further targets
	team := map[string]any{"zillow": map[string]any{"brokerage": "T", "team": map[string]any{"members": []any{
		map[string]any{"profile_url": "https://www.zillow.com/profile/m1", "name": "M1"},
		map[string]any{"profile_url": "", "name": "no url"}}}}}
	if ms := zillowTeamMembers(team); len(ms) != 1 || mStr(ms[0], "profile_url") != "https://www.zillow.com/profile/m1" {
		t.Fatalf("team members wrong: %v", ms)
	}
}

func TestZillowCampaignCreateAndDedupQueue(t *testing.T) {
	root := t.TempDir()
	clientDir := filepath.Join(root, "clients", "acme", "biz_loc", "outreach")
	store := newCrmStore(clientDir)
	if err := store.initTree(); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.createCampaign("z-la", map[string]any{
		"channel_strategy": zillowChannel,
		"zillow_locations": []any{"https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=", "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=kim&page=2"},
		"zillow_keywords":  "kim, Nguyen ,tran,kim",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mStr(cfg, "status") != "paused" {
		t.Fatalf("zillow campaign must be created paused: %q", mStr(cfg, "status"))
	}
	if locs := mList(cfg, "zillow_locations"); len(locs) != 1 {
		t.Fatalf("location variants must dedupe to one clean url: %v", locs)
	}
	if kws := mList(cfg, "zillow_keywords"); len(kws) != 3 || kws[1] != "nguyen" {
		t.Fatalf("keywords must be lower/trim/dedup: %v", kws)
	}
	// Cards dedupe through the client-wide seen registry (a kept FB friend on Zillow too).
	nowStr := nowISO()
	_, _ = withSeen(clientDir, func(reg *seenRegistry) error {
		reg.Profiles["zillow.com/profile/known"] = seenProfile{UID: "zillow.com/profile/known", Status: "kept", UpdatedAt: nowStr}
		return nil
	})
	reg, _ := withSeen(clientDir, func(*seenRegistry) error { return nil })
	fresh, already := zillowCardsToQueue([]map[string]any{
		{"profile_url": "https://www.zillow.com/profile/Known", "name": "K"},
		{"profile_url": "https://www.zillow.com/profile/New%20Agent", "name": "N", "brokerage": "B", "rating": 4.9, "is_team": true},
	}, "seed", nowStr, reg.Profiles, "z-la")
	if already != 1 || len(fresh) != 1 || fresh[0].Subtitle != "B · 4.9★ · TEAM" {
		t.Fatalf("card dedupe/queue wrong: fresh=%+v already=%d", fresh, already)
	}
	// Non-zillow campaigns don't get zillow keys; a group url is refused as a location.
	if _, err := store.createCampaign("bad", map[string]any{"channel_strategy": zillowChannel,
		"zillow_locations": []any{"https://www.zillow.com/homes/la/"}}); err == nil {
		t.Fatal("non-directory url must be refused")
	}
	_ = time.Now()
}

func TestZillowCampaignTemplateAndStatusCLI(t *testing.T) {
	root := t.TempDir()
	clientDir := filepath.Join(root, "clients", "acme", "biz_loc", "outreach")
	store := newCrmStore(clientDir)
	_ = store.initTree()
	if _, err := store.createCampaign("z-la", map[string]any{"channel_strategy": zillowChannel,
		"zillow_locations": []any{"https://www.zillow.com/professionals/real-estate-agent-reviews/example-city-ca/?name="},
		"zillow_keywords":  "kim, nguyen"}); err != nil {
		t.Fatal(err)
	}
	code := 0
	out := captureStdout(t, func() {
		code = runCrmStoreCLI([]string{"--client-dir", clientDir, "harvest", "status", "--campaign", "z-la"})
	})
	if code != 0 || !containsAll(out, `"channel": "zillow_harvest"`, `"zillow_keywords"`) {
		t.Fatalf("zillow harvest status: %d %s", code, out)
	}
	// Template renders the Zillow card + progress with a status map.
	b := &bridge{uiDataRoot: root}
	cfg := store.getCampaign("z-la")
	p, _ := withProgress(clientDir, "z-la", func(p *harvestProgress) error { p.zillow(); return nil })
	hc := harvestConfigFrom(cfg, defaultSystemSettings())
	hc.Channel, hc.Zillow = zillowChannel, zillowConfigFrom(cfg)
	st := b.harvestCommonStatus(p, hc, clientDir)
	st["walk_state"], st["location"], st["keyword"], st["page"] = "walking", hc.Zillow.Locations[0], "kim", 1
	st["queries_done"], st["queries_total"], st["cards_seen"], st["blocked"] = 0, 2, 0, ""
	var sb strings.Builder
	err := uiTpl.ExecuteTemplate(&sb, "campaign", map[string]any{
		"Title": "z-la", "NavPage": "campaign", "Client": uiClient{Slug: "acme", Path: filepath.Dir(clientDir)}, "Slug": "z-la",
		"Status": "paused", "Channel": zillowChannel, "Channels": sortedChannelStrategies(),
		"ZillowLocations": hc.Zillow.Locations, "ZillowKeywords": "kim, nguyen", "ZillowStatus": st,
		"HarvestDefaults": map[string]any{"daily": 500}, "HarvestDaily": "",
		"Goal": map[string]any{}, "Bank": []any{}, "Segment": "", "Sendboxes": []any{}, "Groups": []map[string]any{},
		"Seeds": []string{}, "HarvestKeywords": "", "HarvestStatus": nil,
	})
	if err != nil {
		t.Fatalf("campaign template with zillow: %v", err)
	}
	html := sb.String()
	for _, want := range []string{"Zillow directory", "Leads From Zillow", "agent cards listed", "Start harvest", "example-city-ca"} {
		if !strings.Contains(html, want) {
			t.Errorf("zillow campaign page missing %q", want)
		}
	}
}
