package main

// harvest_zillow.go — "Leads From Zillow": the second source for the harvest
// daemon. Same machine as Leads From Friends (harvest_daemon.go: one action per
// tick, ledger-driven collector rotation, 20-40s pacing, per-box caps, quiet
// hours, circuit breaker + wake amnesty, stale → cancel + re-queue, all state
// on disk under flock) — only the LIST source and the CRM step differ:
//
//   list   = zillow.agents.list on  <location_url> + ?name=<keyword>&page=N
//            (caller-owned pagination; page ends on status "empty", has_more
//            false, or page >= page_count — Zillow serves at most 25 pages per
//            query, so coverage comes from more keywords/locations, not depth)
//   enrich = zillow.profile.enrich on each new card's profile_url
//   CRM    = the DAEMON writes the contact itself: the record is deterministic
//            (__NEXT_DATA__), industry is a fact of the source, so no agent
//            judgement — operator ruling: email OR phone → contact add
//            (match-or-create), neither → skip. Never await_decision.
//
// Cursor: (location index, keyword index, page). Advance page → keyword →
// location; every card is deduped through the client-wide seen registry
// (uid = sourceUID(profile_url), encoded_zuid as a secondary key), team cards
// contribute their members as further enrich targets.
//
// Bot check: the collector's human gate handles it (chime, wait ≤ 6h,
// heartbeat). The daemon's only duty is to NOT treat a job that is waiting for
// the operator as stale, and to quarantine a box only on a FINAL blocked
// record (ceiling passed) — that failure is account-side, wake amnesty does
// not lift it.

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const zillowChannel = "zillow_harvest"

// zillowConfig is the channel's slice of the campaign config.
type zillowConfig struct {
	Locations []string // directory urls, one per location (clean, no page/name)
	Keywords  []string // lower-cased, deduped
}

func zillowConfigFrom(cfg map[string]any) zillowConfig {
	var zc zillowConfig
	seen := map[string]bool{}
	for _, l := range mList(cfg, "zillow_locations") {
		if u := strings.TrimSpace(sprint(l)); u != "" && !seen[u] {
			seen[u] = true
			zc.Locations = append(zc.Locations, u)
		}
	}
	kseen := map[string]bool{}
	for _, k := range mList(cfg, "zillow_keywords") {
		if s := strings.ToLower(strings.TrimSpace(sprint(k))); s != "" && !kseen[s] {
			kseen[s] = true
			zc.Keywords = append(zc.Keywords, s)
		}
	}
	return zc
}

// zillowDirectoryURL validates + normalizes a pasted directory url: must be a
// zillow.com/professionals/... page; `name` and `page` params are STRIPPED
// (the walker sets them), other Zillow filters are kept (they are part of the
// query the operator chose).
func zillowDirectoryURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty url")
	}
	if !schemeRe.MatchString(strings.ToLower(raw)) {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("not a url: %q", raw)
	}
	host := strings.ToLower(u.Host)
	if host != "zillow.com" && host != "www.zillow.com" {
		return "", fmt.Errorf("%q is not a zillow.com url", raw)
	}
	if !strings.HasPrefix(u.Path, "/professionals/") {
		return "", fmt.Errorf("%q is not a Zillow professionals directory url (expected zillow.com/professionals/...)", raw)
	}
	q := u.Query()
	q.Del("name")
	q.Del("page")
	u.RawQuery = q.Encode()
	u.Fragment = ""
	u.Host = "www.zillow.com"
	u.Scheme = "https"
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return u.String(), nil
}

// zillowPageURL composes the leg url: location + keyword + page. Built with the
// url parser, never string concatenation, so a pasted `?name=` / `?name=abc` /
// bare url all come out identical.
func zillowPageURL(location, keyword string, page int) string {
	u, err := url.Parse(location)
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("name", keyword)
	q.Set("page", fmt.Sprint(page))
	u.RawQuery = q.Encode()
	return u.String()
}

// zillowCursor lives in progress.json under "zillow".
type zillowQueryError struct {
	Location string `json:"location"`
	Keyword  string `json:"keyword"`
	Page     int    `json:"page"`
	Reason   string `json:"reason"`
	At       string `json:"at"`
}

type zillowCursor struct {
	Errors     []zillowQueryError `json:"errors,omitempty"` // queries skipped after repeated leg failures
	LocIdx     int            `json:"loc_idx"`
	KwIdx      int            `json:"kw_idx"`
	Page       int            `json:"page"`
	Exhausted  bool           `json:"exhausted"`
	QueriesDone map[string]int `json:"queries_done"` // "loc|kw" -> pages read
	CardsSeen  int            `json:"cards_seen"`
	LastLegURL string         `json:"last_leg_url,omitempty"`
	LastLegAt  string         `json:"last_leg_at,omitempty"`
	LastLegBox string         `json:"last_leg_box,omitempty"`
	LegFailures int           `json:"leg_failures"`
	Blocked    map[string]string `json:"blocked,omitempty"` // box -> ISO time of final blocked record
}

const zillowLegGiveUp = 4 // consecutive failed legs on one query before it is skipped + recorded

// zillowBoxBlocked reports whether the box hit a FINAL Zillow bot-check block
// TODAY (the map is timestamped; older entries are stale and ignored/cleared).
func (c *zillowCursor) zillowBoxBlocked(box string, now time.Time) bool {
	if c == nil || c.Blocked == nil {
		return false
	}
	ts, ok := c.Blocked[box]
	if !ok {
		return false
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil || t.Format("2006-01-02") != now.Format("2006-01-02") {
		delete(c.Blocked, box)
		return false
	}
	return true
}

func (p *harvestProgress) zillow() *zillowCursor {
	if p.Zillow == nil {
		p.Zillow = &zillowCursor{Page: 1, QueriesDone: map[string]int{}}
	}
	if p.Zillow.Page == 0 {
		p.Zillow.Page = 1
	}
	if p.Zillow.QueriesDone == nil {
		p.Zillow.QueriesDone = map[string]int{}
	}
	return p.Zillow
}

// currentQuery returns the leg url for the cursor, or "" when the walk is done.
func (zc zillowConfig) currentQuery(c *zillowCursor) (loc, kw, legURL string) {
	if c.Exhausted || len(zc.Locations) == 0 || len(zc.Keywords) == 0 {
		return "", "", ""
	}
	if c.LocIdx >= len(zc.Locations) {
		c.Exhausted = true
		return "", "", ""
	}
	if c.KwIdx >= len(zc.Keywords) {
		c.KwIdx = 0
		c.LocIdx++
		c.Page = 1
		return zc.currentQuery(c)
	}
	loc, kw = zc.Locations[c.LocIdx], zc.Keywords[c.KwIdx]
	return loc, kw, zillowPageURL(loc, kw, c.Page)
}

// advanceAfterLeg moves the cursor after a leg was read. endOfQuery is true
// when this page was the last for (location, keyword).
func (zc zillowConfig) advanceAfterLeg(c *zillowCursor, endOfQuery bool) {
	key := fmt.Sprintf("%d|%d", c.LocIdx, c.KwIdx)
	c.QueriesDone[key] = c.Page
	if !endOfQuery {
		c.Page++
		return
	}
	c.KwIdx++
	c.Page = 1
	if c.KwIdx >= len(zc.Keywords) {
		c.KwIdx = 0
		c.LocIdx++
	}
	if c.LocIdx >= len(zc.Locations) {
		c.Exhausted = true
	}
}

// zillowLegOutcome extends legOutcome with the directory envelope facts.
type zillowLegFacts struct {
	Status       string // ok | empty | no_next_data | no_display_user | blocked | error
	HasMore      bool
	HasMoreKnown bool // the envelope carried has_more at all
	Page         int
	PageCount    int
}

// zillowEndOfQuery decides whether the (location, keyword) walk ends here:
// an empty page, an explicit has_more=false, or page >= page_count. A missing
// has_more (degraded DOM read) does NOT end the query on its own — the walker
// keeps paging until a page comes back empty (the operator's own rule).
const zillowMaxPagesPerQuery = 25 // Zillow serves at most 25 pages per query (ZILLOW_CAPABILITIES §3)

func zillowEndOfQuery(f zillowLegFacts, cards int, cursorPage int) bool {
	if f.Status == "no_next_data" && cards == 0 {
		return false // degraded read, not an answer — the leg is retried, never taken as the end
	}
	if f.Status == "empty" || cards == 0 {
		return true
	}
	if f.HasMoreKnown && !f.HasMore {
		return true
	}
	if f.PageCount > 0 && f.Page >= f.PageCount {
		return true
	}
	if cursorPage >= zillowMaxPagesPerQuery {
		return true
	}
	return false
}

// zillowCardsToQueue converts agent cards into queue entries (team cards are
// queued too — the enrich record lists members, which are queued afterwards).
func zillowCardsToQueue(items []map[string]any, seedURL, now string, seen map[string]seenProfile, campaign string) (fresh []harvestQueued, already int) {
	for _, it := range items {
		pu := mStr(it, "profile_url")
		uid, _ := sourceUID(pu)
		if uid == "" {
			continue
		}
		if sp, dup := seen[uid]; dup && (terminalSeen(sp.Status) || sp.Campaign == campaign) {
			already++
			continue
		}
		clean, _ := canonicalStoreURL(pu)
		sub := mStr(it, "brokerage")
		if r := it["rating"]; r != nil {
			sub = strings.TrimSpace(sub + fmt.Sprintf(" · %v★", r))
		}
		if t, ok := it["is_team"].(bool); ok && t {
			sub = strings.TrimSpace(sub + " · TEAM")
		}
		fresh = append(fresh, harvestQueued{UID: uid, URL: clean, Name: mStr(it, "name"), Subtitle: sub,
			Seed: seedURL, Priority: 1, QueuedAt: now})
	}
	return fresh, already
}

// zillowContactFields maps a zillow.profile.enrich record to the contact-add
// payload. ok=false when the record has neither email nor phone (operator
// ruling: skip those). Deterministic — no judgement.
func zillowContactFields(rec map[string]any, profileURL, campaign, seedURL string) (map[string]any, bool) {
	emails := strList(rec["emails"])
	phones := strList(rec["phones"])
	if len(emails) == 0 && len(phones) == 0 {
		return nil, false
	}
	z := mMap(rec, "zillow")
	name := mStr(rec, "name")
	given := ""
	if parts := strings.Fields(name); len(parts) > 0 {
		given = parts[0]
	}
	entity := "person"
	if types := strList(z["profile_types"]); len(types) > 0 {
		for _, t := range types {
			if strings.Contains(strings.ToLower(t), "team") {
				entity = "company"
			}
		}
	}
	if tm := mMap(z, "team"); tm != nil && strings.EqualFold(mStr(tm, "role"), "lead") && mInt(tm, "member_count", 0) > 0 {
		// a team LEAD is still a person; the team page itself is a company
	}
	socials := map[string]any{"zillow": strOr(mStr(rec, "profile_url"), profileURL)}
	for k, v := range mMap(z, "socials") {
		if s := strings.TrimSpace(sprint(v)); s != "" && s != "<nil>" {
			socials[k] = s
		}
	}
	// Identity items MUST be maps ({address}/{number}) — the store's identity
	// index and merge read only that shape (leads.go leadToContactFields); a
	// plain string is silently dropped, which would create un-mailable contacts.
	ident := map[string]any{"socials": socials}
	if len(emails) > 0 {
		var list []any
		for i, e := range emails {
			list = append(list, map[string]any{"address": e, "source": "zillow_harvest", "status": "unverified", "is_primary": i == 0})
		}
		ident["emails"] = list
	}
	if len(phones) > 0 {
		var list []any
		for _, ph := range phones {
			list = append(list, map[string]any{"number": ph, "type": "cell", "source": "zillow_harvest"})
		}
		ident["phones"] = list
	}
	if w := strOr(mStr(rec, "website"), mStr(z, "website")); w != "" {
		ident["website"] = w
	}
	custom := map[string]any{"source": "zillow_harvest", "industry": strOr(mStr(rec, "industry"), "Real Estate"),
		"harvest_campaign": campaign, "harvest_seed": seedURL}
	if b := mStr(z, "brokerage"); b != "" {
		custom["brokerage"] = b
	}
	if locs := strList(rec["location"]); len(locs) > 0 {
		custom["location"] = locs[0]
	}
	if zuid := mStr(z, "encoded_zuid"); zuid != "" {
		custom["zillow_zuid"] = zuid
	}
	if cat := mStr(rec, "category"); cat != "" {
		custom["title"] = cat
	}
	return map[string]any{
		"name":          map[string]any{"full": name, "given": given, "entity_type": entity},
		"identities":    ident,
		"tags":          []any{"zillow_directory", "zillow_harvest"},
		"custom_fields": custom,
	}, true
}

// zillowTeamMembers pulls further enrich targets out of a team profile record.
func zillowTeamMembers(rec map[string]any) []map[string]any {
	z := mMap(rec, "zillow")
	tm := mMap(z, "team")
	var out []map[string]any
	for _, m := range asSlice(tm["members"]) {
		if mm, ok := m.(map[string]any); ok && mStr(mm, "profile_url") != "" {
			out = append(out, map[string]any{"profile_url": mStr(mm, "profile_url"), "name": mStr(mm, "name"),
				"brokerage": mStr(z, "brokerage"), "encoded_zuid": mStr(mm, "encoded_zuid")})
		}
	}
	return out
}

// --- daemon hooks (called from harvestCollectResults) --------------------------

// harvestZillowLegDone folds a finished directory page: cards → queue (deduped
// through the client-wide registry), cursor advance, ledger success/failure.
// A blocked (final) record quarantines the box account-side and re-schedules
// the same page on another box; an error keeps the cursor and retries.
func (b *bridge) harvestZillowLegDone(now time.Time, outreachDir, campaign, tag string, f harvestInFlight, out legOutcome, hc harvestConfig) {
	nowStr := nowISO()
	seenReg, _ := withSeen(outreachDir, func(*seenRegistry) error { return nil })
	seen := map[string]seenProfile{}
	if seenReg != nil {
		seen = seenReg.Profiles
	}
	fresh, already := zillowCardsToQueue(out.Items, f.Seed, nowStr, seen, campaign)

	_, _ = withLedger(b.uiDataRoot, now, func(l *harvestLedger) error {
		if out.Failed {
			l.box(f.Box).recordFailure(now, out.Reason)
			if out.Reason == "blocked_bot_check" {
				// Final block: bench this account for Zillow for the day, whatever the counter says.
				l.box(f.Box).QuarantinedUntil = now.Add(ledgerQuarantine).UTC().Format(time.RFC3339)
			}
		} else {
			l.box(f.Box).recordSuccess()
		}
		return nil
	})
	degraded := out.Zillow.Status == "no_next_data" && len(out.Items) == 0
	_, _ = withProgress(outreachDir, campaign, func(p *harvestProgress) error {
		delete(p.InFlight, tag)
		zc := p.zillow()
		if out.Failed || degraded {
			zc.LegFailures++
			zc.LastLegAt, zc.LastLegBox = nowStr, f.Box
			p.Totals["leg_failures"]++
			reason := strOr(out.Reason, "no_next_data (degraded read, 0 cards)")
			if out.Reason == "blocked_bot_check" {
				if zc.Blocked == nil {
					zc.Blocked = map[string]string{}
				}
				zc.Blocked[f.Box] = nowStr
				p.Totals["zillow_blocked"]++
			}
			// A query page that keeps failing across boxes is recorded and SKIPPED —
			// the walk must not stall forever on one broken page.
			if zc.LegFailures >= zillowLegGiveUp {
				loc, kw, _ := hc.Zillow.currentQuery(zc)
				zc.Errors = append(zc.Errors, zillowQueryError{Location: loc, Keyword: kw, Page: zc.Page, Reason: reason, At: nowStr})
				p.Totals["zillow_queries_skipped"]++
				zc.LegFailures = 0
				hc.Zillow.advanceAfterLeg(zc, true)
			}
			return nil
		}
		zc.LegFailures = 0
		if zc.Blocked != nil {
			delete(zc.Blocked, f.Box) // a clean read on this box means the check was passed
		}
		zc.CardsSeen += len(out.Items)
		p.Totals["legs"]++
		p.Totals["friends_seen"] += len(out.Items) // "listed" tile
		p.Totals["already_known"] += already
		p.Queue = append(p.Queue, fresh...)
		hc.Zillow.advanceAfterLeg(zc, zillowEndOfQuery(out.Zillow, len(out.Items), zc.Page))
		return nil
	})
	if len(fresh) > 0 {
		_, _ = withSeen(outreachDir, func(reg *seenRegistry) error {
			for _, q := range fresh {
				reg.Profiles[q.UID] = seenProfile{UID: q.UID, URL: q.URL, Name: q.Name, Status: "queued",
					Seed: q.Seed, Campaign: campaign, FirstSeen: nowStr, UpdatedAt: nowStr}
			}
			return nil
		})
	}
}

// harvestZillowEnrichDone folds a finished profile read: email or phone →
// contact add (match-or-create) → seen kept with lead_id; neither → seen
// rejected ("no email or phone"); failure → retry elsewhere / give up as with FB.
func (b *bridge) harvestZillowEnrichDone(now time.Time, c uiClient, outreachDir, campaign, tag string, f harvestInFlight, out legOutcome) {
	uid := tag
	transient := out.Failed && f.Attempts+1 < harvestMaxAttempts && (isTransient(out.Reason) || out.Reason == "blocked_bot_check")
	if out.Failed && out.Reason == "blocked_bot_check" {
		_, _ = withLedger(b.uiDataRoot, now, func(l *harvestLedger) error {
			l.box(f.Box).recordFailure(now, out.Reason)
			l.box(f.Box).QuarantinedUntil = now.Add(ledgerQuarantine).UTC().Format(time.RFC3339)
			return nil
		})
		_, _ = withProgress(outreachDir, campaign, func(p *harvestProgress) error {
			zc := p.zillow()
			if zc.Blocked == nil {
				zc.Blocked = map[string]string{}
			}
			zc.Blocked[f.Box] = nowISO()
			p.Totals["zillow_blocked"]++
			return nil
		})
	}
	if transient {
		b.harvestFail(now, outreachDir, campaign, tag, f, out.Reason)
		return
	}
	_, _ = withLedger(b.uiDataRoot, now, func(l *harvestLedger) error {
		if out.Failed {
			l.box(f.Box).recordFailure(now, out.Reason)
		} else {
			l.box(f.Box).recordSuccess()
		}
		return nil
	})

	status, leadID, reason := "enrich_failed", "", strOr(out.Reason, "no_record")
	var rec map[string]any
	if !out.Failed && len(out.Items) > 0 {
		_, _ = withProgress(outreachDir, campaign, func(p *harvestProgress) error {
			if zc := p.zillow(); zc.Blocked != nil {
				delete(zc.Blocked, f.Box) // clean read → the box is fine on Zillow again
			}
			return nil
		})
		rec = out.Items[0]
		store := newCrmStore(outreachDir)
		if fields, ok := zillowContactFields(rec, f.URL, campaign, f.Seed); ok {
			id, outcome, err := store.addContact(fields)
			if err != nil {
				status, reason = "enrich_failed", "contact add: "+err.Error()
			} else {
				status, leadID = "kept", id
				reason = "zillow profile with " + zillowReach(rec) + " → contact " + outcome
			}
		} else {
			status, reason = "rejected", "no email or phone published on Zillow (operator rule: skip)"
		}
		// Team profile: its members are further targets.
		if members := zillowTeamMembers(rec); len(members) > 0 {
			seenReg, _ := withSeen(outreachDir, func(*seenRegistry) error { return nil })
			seenMap := map[string]seenProfile{}
			if seenReg != nil {
				seenMap = seenReg.Profiles
			}
			fresh, _ := zillowCardsToQueue(members, f.URL, nowISO(), seenMap, campaign)
			if len(fresh) > 0 {
				_, _ = withProgress(outreachDir, campaign, func(p *harvestProgress) error {
					p.Queue = append(p.Queue, fresh...)
					p.Totals["team_members_queued"] += len(fresh)
					return nil
				})
				_, _ = withSeen(outreachDir, func(reg *seenRegistry) error {
					for _, q := range fresh {
						reg.Profiles[q.UID] = seenProfile{UID: q.UID, URL: q.URL, Name: q.Name, Status: "queued",
							Seed: q.Seed, Campaign: campaign, FirstSeen: nowISO(), UpdatedAt: nowISO()}
					}
					return nil
				})
			}
		}
	}
	// Park the envelope for the drill-down (kept/rejected/failed all keep it), then decide.
	env := map[string]any{"uid": uid, "profile_url": f.URL, "seed": f.Seed, "campaign": campaign,
		"name": f.Name, "subtitle": f.Subtitle, "attempts": f.Attempts + 1,
		"enriched_at": nowISO(), "run_id": f.RunID, "collector": f.Box, "ok": !out.Failed}
	if rec != nil {
		env["record"] = rec
	}
	if out.Failed {
		env["error"] = reason
	}
	path := harvestEnrichedPath(outreachDir, campaign, uid)
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, []byte(marshalIndentJSON(env)+"\n"), 0o600)
	_, _ = withProgress(outreachDir, campaign, func(p *harvestProgress) error {
		delete(p.InFlight, tag)
		p.Totals["enriched"]++
		p.Totals[status]++
		return nil
	})
	_, _ = withSeen(outreachDir, func(reg *seenRegistry) error {
		sp := reg.Profiles[uid]
		if sp.UID == "" {
			sp = seenProfile{UID: uid, URL: f.URL, Name: f.Name, Seed: f.Seed, Campaign: campaign, FirstSeen: nowISO()}
		}
		sp.Status, sp.LeadID, sp.Reason, sp.UpdatedAt = status, leadID, reason, nowISO()
		if sp.Name == "" && rec != nil {
			sp.Name = mStr(rec, "name")
		}
		reg.Profiles[uid] = sp
		return nil
	})
}

func zillowReach(rec map[string]any) string {
	e, p := len(strList(rec["emails"])) > 0, len(strList(rec["phones"])) > 0
	switch {
	case e && p:
		return "email + phone"
	case e:
		return "email"
	default:
		return "phone"
	}
}
