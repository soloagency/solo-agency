package main

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func harvestFixture(t *testing.T) (clientDir string, store *crmStore) {
	t.Helper()
	root := t.TempDir()
	clientDir = filepath.Join(root, "clients", "acme", "biz_loc", "outreach")
	store = newCrmStore(clientDir)
	if err := store.initTree(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.createCampaign("friends-oc", map[string]any{
		"channel_strategy": harvestChannel,
		"goal":             map[string]any{"description": "Realtors and loan officers in Orange County"},
		"seed_profiles":    []any{"https://www.facebook.com/seed.example.person/", "https://m.facebook.com/seed.example.person?ref=x"},
		"harvest":          map[string]any{"goal_keywords": []any{"realtor", "loan"}, "daily_budget": 3},
	}); err != nil {
		t.Fatal(err)
	}
	return clientDir, store
}

func TestHarvestCampaignCreateNormalizesSeeds(t *testing.T) {
	_, store := harvestFixture(t)
	cfg := store.getCampaign("friends-oc")
	// A harvest campaign is created PAUSED: saving must never start walking Facebook.
	if mStr(cfg, "status") != "paused" {
		t.Fatalf("friend_harvest campaign must be created paused, got %q", mStr(cfg, "status"))
	}
	if mStr(store.getCampaign("friends-oc"), "channel_strategy") != harvestChannel {
		t.Fatal("channel lost")
	}
	seeds := mList(cfg, "seed_profiles")
	if len(seeds) != 1 || seeds[0] != "https://www.facebook.com/seed.example.person" {
		t.Fatalf("seed variants must collapse to one clean url: %v", seeds)
	}
	// A group is refused as a seed.
	if _, err := store.createCampaign("bad", map[string]any{"channel_strategy": harvestChannel,
		"seed_profiles": []any{"https://www.facebook.com/groups/examplegroup"}}); err == nil {
		t.Fatal("group url must be refused as a seed profile")
	}
	// Non-harvest channels don't get harvest keys.
	if _, err := store.createCampaign("plain", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if _, has := store.getCampaign("plain")["seed_profiles"]; has {
		t.Fatal("email campaign must not carry seed_profiles")
	}
}

func TestHarvestSeedSyncIngestDedupDecide(t *testing.T) {
	clientDir, store := harvestFixture(t)
	settings := defaultSystemSettings()
	hc := harvestConfigFrom(store.getCampaign("friends-oc"), settings)
	if hc.DailyBudget != 3 || hc.PerBoxBudget != harvestDefaultPerBox || hc.QuietFrom != "01:00" {
		t.Fatalf("config fallback wrong: %+v", hc)
	}
	p, err := syncSeeds(clientDir, "friends-oc", hc)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Seeds) != 1 || p.Seeds[0].FriendsURL != "https://www.facebook.com/seed.example.person/friends" {
		t.Fatalf("seed sync wrong: %+v", p.Seeds)
	}
	if p.CurrentSeed != "https://www.facebook.com/seed.example.person" {
		t.Fatalf("current seed not chosen: %q", p.CurrentSeed)
	}

	seed := p.Seeds[0].URL
	leg1 := []map[string]any{
		{"url": "https://www.facebook.com/alice.realtor", "name": "Alice", "subtitle": "Realtor at OC Homes"},
		{"url": "https://www.facebook.com/bob.smith", "name": "Bob", "subtitle": "12 mutual friends"},
		{"url": "https://www.facebook.com/seed.example.person", "name": "The seed herself"},
	}
	res, err := ingestLeg(clientDir, "friends-oc", seed, legOutcome{Items: leg1, EndCursor: "CURSOR1", HasNext: true, HasNextKnown: true}, hc.GoalKeywords, "ext-a")
	if err != nil {
		t.Fatal(err)
	}
	if res.NewQueued != 2 || res.AlreadySeen != 1 || res.Prioritized != 1 {
		t.Fatalf("leg1 ingest: %+v (seed must be already_seen, alice prioritized)", res)
	}
	// Leg 2 repeats Bob (mutual friend via another path) + one new — Bob is dropped.
	leg2 := []map[string]any{
		{"url": "https://m.facebook.com/bob.smith/", "name": "Bob"},
		{"url": "https://www.facebook.com/carol.loan", "name": "Carol", "subtitle": "Loan Officer"},
	}
	res, err = ingestLeg(clientDir, "friends-oc", seed, legOutcome{Items: leg2, EndCursor: "CURSOR2", HasNext: false, HasNextKnown: true}, hc.GoalKeywords, "ext-b")
	if err != nil {
		t.Fatal(err)
	}
	if res.NewQueued != 1 || res.AlreadySeen != 1 || !res.Exhausted {
		t.Fatalf("leg2 ingest: %+v", res)
	}
	p, _ = withProgress(clientDir, "friends-oc", func(*harvestProgress) error { return nil })
	if len(p.Queue) != 3 || p.Queue[0].Priority != 0 || p.Queue[1].Priority != 0 {
		t.Fatalf("queue must hold 3 with keyword matches first: %+v", p.Queue)
	}
	if !p.Seeds[0].Exhausted || p.Seeds[0].LegsDone != 2 || p.Seeds[0].FriendsSeen != 5 {
		t.Fatalf("seed cursor state wrong: %+v", p.Seeds[0])
	}

	// Decisions: kept needs a lead id; rejected clears; seen registry reflects both.
	if err := recordDecision(clientDir, "friends-oc", "facebook.com/alice.realtor", "kept", "", ""); err == nil {
		t.Fatal("kept without lead id must be refused")
	}
	if err := recordDecision(clientDir, "friends-oc", "facebook.com/alice.realtor", "kept", "c_alice", "realtor in OC"); err != nil {
		t.Fatal(err)
	}
	if err := recordDecision(clientDir, "friends-oc", "facebook.com/bob.smith", "rejected", "", "no trade signal"); err != nil {
		t.Fatal(err)
	}
	reg, _ := withSeen(clientDir, func(*seenRegistry) error { return nil })
	if reg.Profiles["facebook.com/alice.realtor"].Status != "kept" || reg.Profiles["facebook.com/alice.realtor"].LeadID != "c_alice" {
		t.Fatalf("kept not recorded: %+v", reg.Profiles["facebook.com/alice.realtor"])
	}
	if reg.Profiles["facebook.com/bob.smith"].Status != "rejected" {
		t.Fatalf("rejected not recorded")
	}
	// A third leg re-listing Alice from ANOTHER seed is a no-op (client-wide seen).
	res, _ = ingestLeg(clientDir, "friends-oc", "https://www.facebook.com/other.seed", legOutcome{Items: []map[string]any{
		{"url": "https://www.facebook.com/alice.realtor?fbclid=zzz"}}, HasNext: false, HasNextKnown: true}, nil, "ext-a")
	if res.NewQueued != 0 || res.AlreadySeen != 1 {
		t.Fatalf("client-wide dedup failed: %+v", res)
	}
}

func TestHarvestPacingHelpers(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		g := randomGap(rng)
		if g < 20*time.Second || g > 40*time.Second {
			t.Fatalf("gap out of 20-40s: %v", g)
		}
	}
	mk := func(h, m int) time.Time { return time.Date(2026, 8, 16, h, m, 0, 0, time.Local) }
	if !inQuietHours(mk(3, 0), "01:00", "06:00") || inQuietHours(mk(12, 0), "01:00", "06:00") {
		t.Fatal("simple quiet window wrong")
	}
	if !inQuietHours(mk(23, 30), "23:00", "06:00") || !inQuietHours(mk(2, 0), "23:00", "06:00") || inQuietHours(mk(9, 0), "23:00", "06:00") {
		t.Fatal("midnight-crossing quiet window wrong")
	}
	if inQuietHours(mk(3, 0), "bad", "06:00") {
		t.Fatal("unparseable window must not silence the daemon")
	}

	live := []harvestCollector{{InstanceID: "a"}, {InstanceID: "b"}, {InstanceID: "c"}}
	now := time.Now()
	l := &harvestLedger{Boxes: map[string]*ledgerBox{"b": {DayJobs: 150}}}
	c, ok := pickCollector(live, "a", "", l, now, 150, 0)
	if !ok || c.InstanceID != "c" {
		t.Fatalf("rotation must skip b (at cap) after a: got %v %v", c, ok)
	}
	c, ok = pickCollector(live, "c", "", l, now, 150, 0)
	if !ok || c.InstanceID != "a" {
		t.Fatalf("rotation must wrap to a: got %v", c)
	}
	full := &harvestLedger{Boxes: map[string]*ledgerBox{"a": {DayJobs: 150}, "b": {DayJobs: 150}, "c": {DayJobs: 150}}}
	if _, ok := pickCollector(live, "", "", full, now, 150, 0); ok {
		t.Fatal("all at cap must yield no collector")
	}
	// avoid_box skips the box that just failed the friend (failover to another)...
	c, ok = pickCollector(live, "", "a", &harvestLedger{Boxes: map[string]*ledgerBox{}}, now, 150, 0)
	if !ok || c.InstanceID != "b" {
		t.Fatalf("avoid_box must skip a: got %v", c)
	}
	// ...but it is a preference, not a ban: when the avoided box is the ONLY eligible
	// one (others quarantined), it is used rather than stalling the walk (live bug).
	quar := now.Add(time.Hour).UTC().Format(time.RFC3339)
	onlyA := &harvestLedger{Boxes: map[string]*ledgerBox{"b": {QuarantinedUntil: quar}, "c": {QuarantinedUntil: quar}}}
	c, ok = pickCollector(live, "", "a", onlyA, now, 150, 0)
	if !ok || c.InstanceID != "a" {
		t.Fatalf("avoided box must be used when it is the only eligible one: got %v %v", c, ok)
	}
	// Quarantined box is never picked; pacing gap on the BOX (cross-campaign) is honoured.
	q := &harvestLedger{Boxes: map[string]*ledgerBox{"a": {QuarantinedUntil: now.Add(time.Hour).UTC().Format(time.RFC3339)},
		"b": {LastJobAt: now.Add(-5 * time.Second).UTC().Format(time.RFC3339)}}}
	c, ok = pickCollector(live, "", "", q, now, 150, 30*time.Second)
	if !ok || c.InstanceID != "c" {
		t.Fatalf("quarantined a + paced b must yield c: got %v %v", c, ok)
	}
	if friendsURL("https://www.facebook.com/profile.php?id=100001&fbclid=x") != "https://www.facebook.com/profile.php?id=100001&sk=friends" {
		t.Fatalf("numeric friends url wrong: %s", friendsURL("https://www.facebook.com/profile.php?id=100001&fbclid=x"))
	}
}

func TestHarvestCLIRoundTrip(t *testing.T) {
	clientDir, _ := harvestFixture(t)
	code := 0
	out := captureStdout(t, func() {
		code = runCrmStoreCLI([]string{"--client-dir", clientDir, "harvest", "seed", "--campaign", "friends-oc"})
	})
	if code != 0 {
		t.Fatalf("harvest seed exited %d: %s", code, out)
	}
	recs := filepath.Join(t.TempDir(), "leg.json")
	writeJSONT(t, recs, []any{map[string]any{"url": "https://www.facebook.com/dan.agent", "name": "Dan", "subtitle": "Realtor"}})
	out = captureStdout(t, func() {
		code = runCrmStoreCLI([]string{"--client-dir", clientDir, "harvest", "ingest", "--campaign", "friends-oc",
			"--seed", "https://www.facebook.com/seed.example.person", "--records", recs, "--end-cursor", "C1", "--has-next", "true"})
	})
	if code != 0 {
		t.Fatalf("harvest ingest exited %d: %s", code, out)
	}
	out = captureStdout(t, func() {
		code = runCrmStoreCLI([]string{"--client-dir", clientDir, "harvest", "status", "--campaign", "friends-oc"})
	})
	if code != 0 || !containsAll(out, `"queue_len": 1`, `"daily_budget": 3`) {
		t.Fatalf("harvest status: %d %s", code, out)
	}
	// Wrong channel is refused.
	runCrmStoreCLI([]string{"--client-dir", clientDir, "campaign", "create", "--slug", "email-x"})
	out = captureStdout(t, func() {
		code = runCrmStoreCLI([]string{"--client-dir", clientDir, "harvest", "status", "--campaign", "email-x"})
	})
	if code == 0 {
		t.Fatalf("harvest on an email campaign must fail: %s", out)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, x := range subs {
		if !contains(s, x) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestHarvestLegTriStateAndSeedError(t *testing.T) {
	clientDir, store := harvestFixture(t)
	hc := harvestConfigFrom(store.getCampaign("friends-oc"), defaultSystemSettings())
	p, _ := syncSeeds(clientDir, "friends-oc", hc)
	seed := p.Seeds[0].URL
	// A real leg with a cursor and MORE pages: not exhausted.
	ingestLeg(clientDir, "friends-oc", seed, legOutcome{Items: []map[string]any{{"url": "https://www.facebook.com/f1"}},
		EndCursor: "C1", HasNext: true, HasNextKnown: true}, nil, "ext-a")
	// A failed leg (no signal): cursor kept, NOT exhausted, failure counted, box remembered.
	res, _ := ingestLeg(clientDir, "friends-oc", seed, legOutcome{Failed: true, Reason: "capture_timeout"}, nil, "ext-b")
	if res.Exhausted || !res.LegFailed {
		t.Fatalf("failed leg must not exhaust: %+v", res)
	}
	p, _ = withProgress(clientDir, "friends-oc", func(*harvestProgress) error { return nil })
	if p.Seeds[0].Exhausted || p.Seeds[0].EndCursor != "C1" || p.Seeds[0].LegFailures != 1 || !hasStr(p.Seeds[0].TriedBoxes, "ext-b") {
		t.Fatalf("seed state after failed leg wrong: %+v", p.Seeds[0])
	}
	// A leg with items but NO has_next signal at all: also not exhausted (unknown ≠ end).
	ingestLeg(clientDir, "friends-oc", seed, legOutcome{Items: []map[string]any{{"url": "https://www.facebook.com/f2"}}, EndCursor: "C2"}, nil, "ext-a")
	p, _ = withProgress(clientDir, "friends-oc", func(*harvestProgress) error { return nil })
	if p.Seeds[0].Exhausted || p.Seeds[0].EndCursor != "C2" {
		t.Fatalf("unknown has_next must not exhaust, cursor must advance: %+v", p.Seeds[0])
	}
	// Only an explicit has_next_page=false ends the walk.
	ingestLeg(clientDir, "friends-oc", seed, legOutcome{Items: nil, EndCursor: "C3", HasNext: false, HasNextKnown: true}, nil, "ext-a")
	p, _ = withProgress(clientDir, "friends-oc", func(*harvestProgress) error { return nil })
	if !p.Seeds[0].Exhausted {
		t.Fatalf("explicit has_next=false must exhaust: %+v", p.Seeds[0])
	}

	// A seed that returns NOTHING on two different boxes before any real leg -> Error, not done.
	if _, err := store.campaignUpdate("friends-oc", map[string]any{"seed_profiles": []any{
		"https://www.facebook.com/seed.example.person", "https://www.facebook.com/private.person"}}); err != nil {
		t.Fatal(err)
	}
	hc = harvestConfigFrom(store.getCampaign("friends-oc"), defaultSystemSettings())
	_, _ = withProgress(clientDir, "friends-oc", func(p *harvestProgress) error {
		reconcileSeeds(p, hc, clientDir, "friends-oc")
		return nil
	})
	priv := "https://www.facebook.com/private.person"
	ingestLeg(clientDir, "friends-oc", priv, legOutcome{Failed: true, Reason: "no_record"}, nil, "ext-a")
	res, _ = ingestLeg(clientDir, "friends-oc", priv, legOutcome{Failed: true, Reason: "no_record"}, nil, "ext-b")
	if res.SeedError == "" {
		t.Fatalf("seed unreadable on 2 boxes must surface an error: %+v", res)
	}
	p, _ = withProgress(clientDir, "friends-oc", func(*harvestProgress) error { return nil })
	if p.Seeds[1].Exhausted || p.Seeds[1].Error == "" {
		t.Fatalf("private seed must be flagged, never 'done': %+v", p.Seeds[1])
	}
	// Removing a seed from the config flags it Removed on reconcile, cursor kept.
	store.campaignUpdate("friends-oc", map[string]any{"seed_profiles": []any{"https://www.facebook.com/seed.example.person"}})
	hc = harvestConfigFrom(store.getCampaign("friends-oc"), defaultSystemSettings())
	_, _ = withProgress(clientDir, "friends-oc", func(p *harvestProgress) error {
		reconcileSeeds(p, hc, clientDir, "friends-oc")
		return nil
	})
	p, _ = withProgress(clientDir, "friends-oc", func(*harvestProgress) error { return nil })
	if !p.Seeds[1].Removed || p.Seeds[0].Removed {
		t.Fatalf("reconcile must flag only the dropped seed: %+v", p.Seeds)
	}
}

func TestHarvestRequeueAndLedgerBreaker(t *testing.T) {
	now := time.Now()
	p := &harvestProgress{Totals: map[string]int{}, InFlight: map[string]harvestInFlight{}}
	f := harvestInFlight{URL: "https://www.facebook.com/x", Seed: "s", Box: "ext-a", Name: "X", Attempts: 0}
	if gaveUp := requeueFriend(p, f, "facebook.com/x", "ext-a", "stale"); gaveUp {
		t.Fatal("first failure must requeue, not give up")
	}
	if len(p.Queue) != 1 || p.Queue[0].Attempts != 1 || p.Queue[0].AvoidBox != "ext-a" || p.Queue[0].Priority != 0 {
		t.Fatalf("requeue shape wrong: %+v", p.Queue)
	}
	f.Attempts = harvestMaxAttempts - 1
	if gaveUp := requeueFriend(p, f, "facebook.com/x", "ext-b", "stale"); !gaveUp {
		t.Fatal("must give up at max attempts")
	}
	// Circuit breaker: 3 consecutive failures -> quarantine; success clears.
	bx := &ledgerBox{}
	if bx.recordFailure(now, "t1") || bx.recordFailure(now, "t2") {
		t.Fatal("must not trip before threshold")
	}
	if !bx.recordFailure(now, "t3") || !bx.quarantined(now) {
		t.Fatal("third consecutive failure must quarantine")
	}
	if bx.eligible(now, 150, 0) {
		t.Fatal("quarantined box must be ineligible")
	}
	if !bx.eligible(now.Add(ledgerQuarantine+time.Minute), 150, 0) {
		t.Fatal("quarantine must expire")
	}
	bx.recordSuccess()
	if bx.ConsecutiveFailures != 0 || bx.QuarantinedUntil != "" {
		t.Fatal("success must reset the breaker")
	}
	// Ledger persistence + day rollover.
	root := t.TempDir()
	withLedger(root, now, func(l *harvestLedger) error { l.box("ext-a").recordJob(now, "enrich"); return nil })
	l, _ := withLedger(root, now, func(*harvestLedger) error { return nil })
	if l.Boxes["ext-a"].DayJobs != 1 {
		t.Fatalf("ledger not persisted: %+v", l.Boxes["ext-a"])
	}
	l, _ = withLedger(root, now.Add(48*time.Hour), func(*harvestLedger) error { return nil })
	if l.Boxes["ext-a"].DayJobs != 0 || l.Boxes["ext-a"].TotalJobs != 1 {
		t.Fatalf("day rollover must reset day counters only: %+v", l.Boxes["ext-a"])
	}
	// isTransient classification.
	if !isTransient("capture_timeout") || !isTransient("landed_on_self") || isTransient("profile is private, no about") {
		t.Fatal("transient classification wrong")
	}
}

func TestHarvestDecideCLIValidates(t *testing.T) {
	clientDir, _ := harvestFixture(t)
	code := 0
	// Nothing pending -> decide is refused (prevents a mis-canonicalised url from leaving the real record pending).
	out := captureStdout(t, func() {
		code = runCrmStoreCLI([]string{"--client-dir", clientDir, "harvest", "decide", "--campaign", "friends-oc",
			"--profile", "https://www.facebook.com/nobody", "--status", "rejected", "--reason", "x"})
	})
	if code == 0 {
		t.Fatalf("decide on a non-pending uid must fail: %s", out)
	}
	// Put a record into await_decision and decide it via the CLI with --status (the flag the review found unregistered).
	_, _ = withProgress(clientDir, "friends-oc", func(p *harvestProgress) error {
		p.AwaitDecision = append(p.AwaitDecision, "facebook.com/dan.agent")
		return nil
	})
	out = captureStdout(t, func() {
		code = runCrmStoreCLI([]string{"--client-dir", clientDir, "harvest", "decide", "--campaign", "friends-oc",
			"--profile", "https://www.facebook.com/dan.agent/", "--status", "kept", "--lead-id", "c_dan", "--reason", "realtor OC"})
	})
	if code != 0 {
		t.Fatalf("decide with --status must succeed: %s", out)
	}
	p, _ := withProgress(clientDir, "friends-oc", func(*harvestProgress) error { return nil })
	if len(p.AwaitDecision) != 0 {
		t.Fatalf("decide must clear await_decision: %+v", p.AwaitDecision)
	}
}

func TestHarvestDrilldownRowsAndTemplate(t *testing.T) {
	clientDir, store := harvestFixture(t)
	hc := harvestConfigFrom(store.getCampaign("friends-oc"), defaultSystemSettings())
	p, _ := syncSeeds(clientDir, "friends-oc", hc)
	seed := p.Seeds[0].URL
	ingestLeg(clientDir, "friends-oc", seed, legOutcome{Items: []map[string]any{
		{"url": "https://www.facebook.com/alice.realtor", "name": "Alice", "subtitle": "Realtor at OC Homes"},
		{"url": "https://www.facebook.com/bob.smith", "name": "Bob"},
	}, EndCursor: "C1", HasNext: true, HasNextKnown: true}, hc.GoalKeywords, "ext-a")
	// Park an enriched envelope for Alice and put her in await_decision.
	env := map[string]any{"uid": "facebook.com/alice.realtor", "profile_url": "https://www.facebook.com/alice.realtor",
		"seed": seed, "name": "Alice", "ok": true, "attempts": 1, "collector": "ext-a",
		"record": map[string]any{"category": "Real Estate Agent", "about_lines": []any{"Realtor · OC Homes"},
			"work":   []any{map[string]any{"title": "Realtor", "employer": "OC Homes"}},
			"posts":  []any{map[string]any{"caption": "Just listed in Garden Grove", "date": "2026-08-10", "url": "https://www.facebook.com/alice.realtor/posts/1"}},
			"emails": []any{"alice@example.com"}}}
	writeJSONT(t, harvestEnrichedPath(clientDir, "friends-oc", "facebook.com/alice.realtor"), env)
	_, _ = withProgress(clientDir, "friends-oc", func(p *harvestProgress) error {
		p.Queue = p.Queue[1:] // alice popped
		p.AwaitDecision = append(p.AwaitDecision, "facebook.com/alice.realtor")
		return nil
	})
	recordDecisionOK := recordDecision(clientDir, "friends-oc", "facebook.com/bob.smith", "rejected", "", "no trade signal")
	if recordDecisionOK != nil {
		t.Fatal(recordDecisionOK)
	}
	b := &bridge{}
	c := uiClient{Slug: "acme", Path: filepath.Dir(clientDir)}
	rows := b.harvestRows(c, "friends-oc", "await", time.Now())
	if len(rows) != 1 || rows[0].Name != "Alice" || rows[0].OK != "ok" || len(rows[0].Posts) != 1 || rows[0].Category != "Real Estate Agent" {
		t.Fatalf("await drill-down wrong: %+v", rows)
	}
	if rows := b.harvestRows(c, "friends-oc", "rejected", time.Now()); len(rows) != 1 || rows[0].Reason != "no trade signal" {
		t.Fatalf("rejected drill-down wrong: %+v", rows)
	}
	if rows := b.harvestRows(c, "friends-oc", "known", time.Now()); len(rows) < 2 { // seed + bob
		t.Fatalf("known drill-down should include the seed and decided profiles: %+v", rows)
	}
	var sb strings.Builder
	err := uiTpl.ExecuteTemplate(&sb, "harvest", map[string]any{
		"Title": "x", "NavPage": "harvest", "Client": c, "Slug": "friends-oc", "Stage": "await",
		"StageLabel": "Awaiting decision", "StageHelp": "h", "Stages": harvestStages, "Rows": rows, "Count": len(rows),
		"Goal": "Realtors in OC"})
	if err != nil {
		t.Fatalf("harvest template: %v", err)
	}
	html := sb.String()
	for _, want := range []string{"Alice", "Real Estate Agent", "Just listed in Garden Grove", "alice@example.com", "Realtors in OC"} {
		if !strings.Contains(html, want) {
			t.Errorf("harvest page missing %q", want)
		}
	}
}

func TestHarvestWakeAmnesty(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	withLedger(root, now, func(l *harvestLedger) error {
		a := l.box("ext-a")
		a.ConsecutiveFailures, a.LastError = 3, "job stale after 20m (never claimed)"
		a.QuarantinedUntil = now.Add(time.Hour).UTC().Format(time.RFC3339)
		b := l.box("ext-b")
		b.ConsecutiveFailures, b.LastError = 3, "landed_on_self"
		b.QuarantinedUntil = now.Add(time.Hour).UTC().Format(time.RFC3339)
		return nil
	})
	br := &bridge{uiDataRoot: root}
	br.harvestWakeAmnesty(now, now.Add(-10*time.Minute))
	l, _ := withLedger(root, now, func(*harvestLedger) error { return nil })
	if l.Boxes["ext-a"].quarantined(now) || l.Boxes["ext-a"].ConsecutiveFailures != 0 {
		t.Fatalf("machine-side quarantine must be lifted on wake: %+v", l.Boxes["ext-a"])
	}
	if !l.Boxes["ext-b"].quarantined(now) {
		t.Fatalf("account-side quarantine (landed_on_self) must survive wake: %+v", l.Boxes["ext-b"])
	}
	if !isMachineSideFailure("no_record") || !isMachineSideFailure("source error") || isMachineSideFailure("account checkpoint") {
		t.Fatal("machine-side classification wrong")
	}
}

// TestCancelPendingJobIsProofNotGuess: withdrawing a stale job must be decided by
// os.Remove, never by a preceding os.Stat. The extension claims a job by renaming
// the file, and it polls every ~5s, so a Stat-then-Remove pair loses the race often
// enough to matter — and the two outcomes are opposites: a job that was never
// claimed cannot write anything, while a claimed one still can.
func TestCancelPendingJobIsProofNotGuess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "job.json")
	if err := os.WriteFile(path, []byte(`{"run_id":"r1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !cancelPendingJob(path) {
		t.Fatal("an unclaimed pending file must be reported as cancelled")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("cancelling must actually remove the pending file")
	}
	// The claim landed first: the file is gone, and this is NOT a cancellation.
	if cancelPendingJob(path) {
		t.Fatal("a file already claimed (renamed away) must not be reported as cancelled")
	}
	if cancelPendingJob("") {
		t.Fatal("no pending path recorded is not a cancellation")
	}
}

// TestCollectorPolicyForJob: the policy stopped being decorative, so the bridge has
// to mint the permission a legitimate write needs — and only that one.
func TestCollectorPolicyForJob(t *testing.T) {
	read := collectorPolicyForJob(map[string]any{"sources": []any{
		map[string]any{"capability": "fb.profile.enrich"},
	}})
	for _, flag := range []string{"do_not_comment", "do_not_post", "do_not_message", "do_not_react", "read_only"} {
		if read[flag] != true {
			t.Fatalf("a read-only job must keep %s=true, got %v", flag, read[flag])
		}
	}
	write := collectorPolicyForJob(map[string]any{"sources": []any{
		map[string]any{"capability": "fb.post.comment"},
	}})
	if write["do_not_comment"] != false {
		t.Fatal("a comment job must arrive with do_not_comment cleared, or the collector refuses it")
	}
	if write["read_only"] != false {
		t.Fatal("a write job is not read_only")
	}
	for _, flag := range []string{"do_not_post", "do_not_message", "do_not_react"} {
		if write[flag] != true {
			t.Fatalf("a comment job must NOT be allowed to %s", flag)
		}
	}
	// Nothing is derived from a capability the job does not actually carry.
	if empty := collectorPolicyForJob(map[string]any{}); empty["do_not_comment"] != true {
		t.Fatal("a job with no sources must be denied every write")
	}
}

// A seed error was a one-way door: nothing cleared Error, and Error is what bars
// a seed from selection, so one collector outage retired the seed permanently.
// `harvest seed` is the operator's way back, and it must not throw away the walk.
func TestHarvestSeedSyncClearsTheBreakerAndKeepsTheCursor(t *testing.T) {
	_, store := harvestFixture(t)
	cfg := store.getCampaign("friends-oc")
	hc := harvestConfigFrom(cfg, loadSystemSettings(pipelineRootFromClientDir(store.clientDir)))
	hc.Channel = harvestChannel
	if _, err := syncSeeds(store.clientDir, "friends-oc", hc); err != nil {
		t.Fatal(err)
	}
	if _, err := withProgress(store.clientDir, "friends-oc", func(p *harvestProgress) error {
		p.Seeds[0].Error = "4 consecutive leg failures: no result"
		p.Seeds[0].LegFailures = 4
		p.Seeds[0].TriedBoxes = []string{"box-a", "box-b"}
		p.Seeds[0].EndCursor = "CURSOR"
		p.CurrentSeed = p.Seeds[0].URL
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	p, err := syncSeeds(store.clientDir, "friends-oc", hc)
	if err != nil {
		t.Fatal(err)
	}
	s := p.Seeds[0]
	if s.Error != "" || s.LegFailures != 0 || len(s.TriedBoxes) != 0 {
		t.Fatalf("harvest seed must clear the breaker, got error=%q failures=%d tried=%v", s.Error, s.LegFailures, s.TriedBoxes)
	}
	if s.EndCursor != "CURSOR" {
		t.Fatalf("a partly walked seed must resume from its cursor, got %q", s.EndCursor)
	}
	// The pointer is not left empty: syncSeeds re-picks straight away, and it must
	// land on a seed that is actually eligible.
	if p.CurrentSeed == "" {
		t.Fatal("current_seed must be re-picked, not left empty")
	}
	for _, s := range p.Seeds {
		if s.URL == p.CurrentSeed && (s.Error != "" || s.Removed || s.Exhausted) {
			t.Fatalf("current_seed points at an ineligible seed: %+v", s)
		}
	}
}

// TestHarvestSeedTally: "no seed left to walk" has two causes and only one is good
// news. A campaign whose seeds all ERRORED read nothing at all; reporting that as
// "walk finished — every seed's friend list was read" is the lie that let a dead
// campaign look healthy for two days on the live install (2026-08-19: four seeds,
// every one carrying an error, 0 enriched that day).
func TestHarvestSeedTally(t *testing.T) {
	cases := []struct {
		name              string
		seeds             []harvestSeed
		blocked           bool
		errored, finished int
		stuck             bool
	}{
		{"all read to the end", []harvestSeed{{Exhausted: true}, {Exhausted: true}},
			true, 0, 2, false},
		{"all failed", []harvestSeed{{Error: "4 consecutive leg failures"}, {Error: "friend list returned nothing"}},
			true, 2, 0, true},
		{"one finished, one failed", []harvestSeed{{Exhausted: true}, {Error: "boom"}},
			true, 1, 1, true},
		{"one still walkable", []harvestSeed{{Error: "boom"}, {}},
			false, 1, 0, false},
		{"removed seeds do not count either way", []harvestSeed{{Removed: true}, {Exhausted: true}},
			true, 0, 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &harvestProgress{Seeds: tc.seeds}
			blocked, errored, finished := harvestSeedTally(p)
			if blocked != tc.blocked || errored != tc.errored || finished != tc.finished {
				t.Fatalf("tally = (%v,%d,%d), want (%v,%d,%d)", blocked, errored, finished,
					tc.blocked, tc.errored, tc.finished)
			}
			if got := harvestSeedsBlockedByError(p); got != tc.stuck {
				t.Fatalf("stuck = %v, want %v", got, tc.stuck)
			}
			// A walk that is genuinely finished must never be reported as stuck, and
			// vice versa — these two states send the operator to opposite actions.
			if harvestSeedsBlockedByError(p) && harvestAllSeedsDone(p) && tc.errored == 0 {
				t.Fatal("a clean finish must not also read as stuck")
			}
		})
	}
	// No seeds at all is neither finished nor stuck: the campaign was never set up.
	if b, _, _ := harvestSeedTally(&harvestProgress{}); b {
		t.Fatal("a campaign with no seeds has not finished a walk")
	}
}

// TestBreakerByFailureKind: the breaker used to count every failure the same and
// bench at three-in-a-row. Measured 2026-08-18 on the live install: the background
// rate of empty reads was ~33%, so three in a row arrived by CHANCE about every 25
// attempts — the campaign spent the day doing ~20 minutes of work then two hours
// benched, ~7 profiles an hour against a budget of 500.
func TestBreakerByFailureKind(t *testing.T) {
	now := time.Now()

	// SOFT failures are noise: they must not bench the box at three in a row.
	soft := &ledgerBox{}
	for i := 0; i < 3; i++ {
		if soft.recordFailure(now, "no_record") {
			t.Fatalf("empty read %d benched the box — that is the bug being fixed", i+1)
		}
	}
	if soft.quarantined(now) {
		t.Fatal("three empty reads must not quarantine")
	}
	// ...they slow it down instead, continuously — once there is enough history to
	// call it a rate rather than a coincidence (ledgerPacingMinN).
	if m := soft.pacingMultiplier(); m != 1 {
		t.Fatalf("three outcomes is too thin a sample to penalise, multiplier=%v", m)
	}
	soft.recordSuccess()
	soft.recordFailure(now, "no_record") // 5 outcomes, 4 of them failures
	if m := soft.pacingMultiplier(); m <= 1 {
		t.Fatalf("a box failing most reads must be paced slower, multiplier=%v", m)
	}
	gap := 30 * time.Second
	soft.LastJobAt = now.Add(-40 * time.Second).UTC().Format(time.RFC3339)
	if soft.eligible(now, 150, gap) {
		t.Fatal("40s after the last job must be too soon for a box failing this often")
	}
	// A clean box at the same moment is fine — the stretch comes from ITS history.
	clean := &ledgerBox{LastJobAt: soft.LastJobAt}
	if !clean.eligible(now, 150, gap) {
		t.Fatal("a healthy box must not be slowed by another box's failures")
	}

	// ACCOUNT-side is a verdict: one is enough, and it is benched for longer.
	acct := &ledgerBox{}
	if !acct.recordFailure(now, "landed_on_self") {
		t.Fatal("an account-side signal must bench immediately, not after three")
	}
	until, _ := time.Parse(time.RFC3339, acct.QuarantinedUntil)
	if until.Sub(now) < ledgerQuarantine {
		t.Fatalf("an account signal must bench longer than a routine trip: %v", until.Sub(now))
	}
	// And wake amnesty must never forgive it — the laptop being asleep is not why
	// Facebook served the operator's own profile.
	if isMachineSideFailure("landed_on_self") || isMachineSideFailure("action_blocked") {
		t.Fatal("account-side failures must not read as machine-side")
	}

	// A box getting nowhere at all still rests, briefly, so it stops burning jobs.
	dead := &ledgerBox{}
	tripped := false
	for i := 0; i < ledgerWindow; i++ {
		if dead.recordFailure(now, "no_record") {
			tripped = true
		}
	}
	if !tripped || !dead.quarantined(now) {
		t.Fatal("a window of nothing but failures must earn a cool-off")
	}
	cool, _ := time.Parse(time.RFC3339, dead.QuarantinedUntil)
	if d := cool.Sub(now); d > ledgerQuarantine {
		t.Fatalf("a cool-off is a rest, not the old two-hour punishment: %v", d)
	}

	// Unclassified failures keep the old behaviour exactly.
	other := &ledgerBox{}
	if other.recordFailure(now, "something nobody classified") ||
		other.recordFailure(now, "something nobody classified") {
		t.Fatal("unknown failures must still take three")
	}
	if !other.recordFailure(now, "something nobody classified") {
		t.Fatal("three unknown failures must still trip the breaker")
	}

	// Success clears the slate and the pacing penalty decays with the window.
	rec := &ledgerBox{}
	for i := 0; i < ledgerWindow; i++ {
		rec.recordFailure(now, "no_record")
	}
	for i := 0; i < ledgerWindow; i++ {
		rec.recordSuccess()
	}
	if m := rec.pacingMultiplier(); m != 1 {
		t.Fatalf("a recovered box must return to normal pace, multiplier=%v", m)
	}
	if rec.quarantined(now) || rec.ConsecutiveFailures != 0 {
		t.Fatal("success must clear the bench and the counter")
	}
}

// The breaker retires a seed after four consecutive failed legs. With a flat
// two-minute rest that only applied to the SAME box, rotating across three
// collectors fired seven legs in five minutes, so a transient Facebook throttle
// filled the counter before it could pass. The rest must grow instead.
func TestHarvestLegRestBacksOffInsteadOfAccelerating(t *testing.T) {
	if got := harvestLegRest(0); got != harvestLegBaseWait {
		t.Fatalf("a healthy seed keeps the base rest, got %v", got)
	}
	prev := harvestLegRest(0)
	for f := 1; f <= 4; f++ {
		got := harvestLegRest(f)
		if got <= prev {
			t.Fatalf("rest must grow with failures: %d -> %v, previous %v", f, got, prev)
		}
		prev = got
	}
	if harvestLegRest(9) != harvestLegRest(4) {
		t.Fatal("rest must be capped so a dead seed does not wait forever")
	}
	// The whole point: four failures must span far more than the few minutes a
	// three-box rotation used to take.
	var total time.Duration
	for f := 1; f <= 3; f++ {
		total += harvestLegRest(f)
	}
	if total < 20*time.Minute {
		t.Fatalf("four consecutive failures still fit in %v, the throttle window wins again", total)
	}
}

// TestSeedBreakerIgnoresWeather: the collector breaker learned to ride out soft
// failures; the seed breaker did not, so the same throttling simply killed the
// campaign one layer down instead. Measured live 2026-08-21: both seeds retired
// on `no result` and `gql_capability_timeout`, one of them after 25 good legs,
// and the campaign needed a human to run `harvest seed` again.
func TestSeedBreakerIgnoresWeather(t *testing.T) {
	clientDir, store := harvestFixture(t)
	hc := harvestConfigFrom(store.getCampaign("friends-oc"), defaultSystemSettings())
	p, err := syncSeeds(clientDir, "friends-oc", hc)
	if err != nil {
		t.Fatal(err)
	}
	seed := p.Seeds[0].URL

	// One good leg first: this seed demonstrably works.
	if _, err := ingestLeg(clientDir, "friends-oc", seed, legOutcome{
		Items:     []map[string]any{{"url": "https://www.facebook.com/example.friend.001", "name": "A"}},
		EndCursor: "C1", HasNext: true, HasNextKnown: true}, nil, "ext-a"); err != nil {
		t.Fatal(err)
	}

	// Then a throttled afternoon: six soft failures in a row.
	for i := 0; i < 6; i++ {
		if _, err := ingestLeg(clientDir, "friends-oc", seed,
			legOutcome{Failed: true, Reason: "no result"}, nil, "ext-a"); err != nil {
			t.Fatal(err)
		}
	}
	p, _ = withProgress(clientDir, "friends-oc", func(*harvestProgress) error { return nil })
	if p.Seeds[0].Error != "" {
		t.Fatalf("a working seed must not be retired by weather, got %q", p.Seeds[0].Error)
	}
	if p.Seeds[0].LegFailures < 4 {
		t.Fatal("the failures must still be counted — that is what makes it rest longer")
	}
	// The rest between attempts grows with those failures, which IS the slowdown.
	if harvestLegRest(p.Seeds[0].LegFailures) <= harvestLegRest(0) {
		t.Fatal("a failing seed must wait longer between legs")
	}

	// A HARD failure still retires it: that is not weather.
	if _, err := ingestLeg(clientDir, "friends-oc", seed,
		legOutcome{Failed: true, Reason: "landed_on_self"}, nil, "ext-a"); err != nil {
		t.Fatal(err)
	}
	p, _ = withProgress(clientDir, "friends-oc", func(*harvestProgress) error { return nil })
	if p.Seeds[0].Error == "" {
		t.Fatal("an account-side failure must still stamp a seed error")
	}

	// And a new day clears an error that was only ever weather, so a throttled
	// afternoon does not cost a manual `harvest seed` tomorrow.
	_, _ = withProgress(clientDir, "friends-oc", func(pp *harvestProgress) error {
		pp.Seeds[0].Error = "4 consecutive leg failures: no result"
		pp.Seeds[0].LegFailures = 4
		pp.DayKey = "1999-01-01"
		return nil
	})
	p, _ = withProgress(clientDir, "friends-oc", func(pp *harvestProgress) error {
		pp.resetDayIfNeeded(time.Now())
		return nil
	})
	if p.Seeds[0].Error != "" || p.Seeds[0].LegFailures != 0 {
		t.Fatalf("a soft seed error must not survive the day rollover: %q", p.Seeds[0].Error)
	}
	// A structural one must survive it.
	_, _ = withProgress(clientDir, "friends-oc", func(pp *harvestProgress) error {
		pp.Seeds[0].Error = "friend list returned nothing on 2 collector(s) — private to those accounts"
		pp.DayKey = "1999-01-01"
		return nil
	})
	p, _ = withProgress(clientDir, "friends-oc", func(pp *harvestProgress) error {
		pp.resetDayIfNeeded(time.Now())
		return nil
	})
	if p.Seeds[0].Error == "" {
		t.Fatal("a structural seed error must survive the rollover — it needs a human")
	}
}
