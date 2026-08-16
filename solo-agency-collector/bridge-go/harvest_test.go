package main

import (
	"math/rand"
	"path/filepath"
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
	// avoid_box skips the box that just failed the friend (failover to another).
	c, ok = pickCollector(live, "", "a", &harvestLedger{Boxes: map[string]*ledgerBox{}}, now, 150, 0)
	if !ok || c.InstanceID != "b" {
		t.Fatalf("avoid_box must skip a: got %v", c)
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

func contains(s, sub string) bool { return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0) }

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
