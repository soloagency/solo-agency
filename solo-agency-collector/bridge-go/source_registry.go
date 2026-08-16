package main

// source_registry.go — cross-client shared-scan bookkeeping.
//
// Many clients in one install monitor the SAME sources (an industry Facebook
// group, a competitor page, a recurring public news site). Until now every
// client's run scanned every source again: with 20 same-industry clients that
// is 20 identical scans a day — hours of Chrome time and, worse, 20× the
// automation footprint on the operator's logged-in accounts. The extension-
// per-client model is deliberate (it spreads scanning across accounts) and is
// NOT changed here. What this module adds is exactly three things:
//
//   1. Source identity: one canonical UID per source (facebook.com/groups/<id>
//      for groups, normalizeSocial() for everything else), so the same group
//      registered by ten clients is ONE entry, not ten unrelated strings.
//   2. Freshness tracking: the registry remembers the last completed scan of
//      each shared source (who scanned, when, where the data landed).
//   3. Check-before-job: `tool source-registry due` tells a run which of its
//      sources actually need scanning and which have fresh data to reuse via
//      the recorded data_dir pointer. Whichever client's run comes first (and
//      whose login can see the source) performs the day's scan; later runs
//      read the pointer instead of re-scanning.
//
// Freshness is a rolling TTL in hours (default 20h ≈ once a day), NEVER a
// calendar-day compare — a 9am scan serves an 11pm run, and the run_window
// (Stage 4) governs what each client CONSUMES independently of when scans
// happened. Client-owned sources (their own page/group/site) are `exclusive`
// and always scan under their own client, exactly as before.
//
// The same check-before-work pattern is exposed for public keyword searches
// (`tool search-pool`): same-industry clients share one day's search results
// for one normalized keyword.
//
// Everything here is a CLI on the shared bridge binary, not bridge server
// state: registry updates are read-modify-write under an exclusive flock so
// concurrent client runs cannot lose each other's writes.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const sourceRegistryDefaultTTLHours = 20.0
const scanClaimTTLHours = 2.0
const searchPoolPruneDays = 7.0

// --- canonical source identity -------------------------------------------------

// Mobile mirror hosts collapse into the main host ONLY for known social
// platforms. A bare `^(m|web|l)\.` strip would mangle real hosts (web.dev,
// m.me) and — worse — fold link REDIRECTORS (l.facebook.com) into the platform
// itself; the redirector's real identity lives in its `u=` param, which
// normalizeSocial rightly drops as non-identity. Redirectors are resolved to
// their destination BEFORE canonicalization, or rejected when opaque.
var socialMirrorPrefixRe = regexp.MustCompile(`^(?:m|mbasic|touch|web)\.((?:facebook|instagram|tiktok|linkedin|reddit|youtube|twitter|x)\.com)`)
var redirectorHostRe = regexp.MustCompile(`^(?:https?://)?(?:www\.)?(?:l|lm)\.(?:facebook|instagram|messenger)\.com(?:/|$)`)
var fbGroupUIDRe = regexp.MustCompile(`^facebook\.com/groups/([^/?]+)`)

// sourceUID canonicalizes a source URL into its cross-client identity key.
// Built on normalizeSocial (the CRM's tested identity canonicalizer: lowercase,
// no scheme/www/fragment/trailing slash, tracking params stripped, identity
// params kept). On top of that: social mobile-mirror hosts collapse into the
// main host, l.facebook.com-style redirectors resolve to their destination,
// and a Facebook group URL collapses to `facebook.com/groups/<id>` whatever
// tab/sub-path/params it was registered with. Websites keep their path (uid =
// normalized URL, not just the domain): two clients pointing at different
// sections of one big site are genuinely different scan targets, while the
// common case — the same URL registered by many — still collapses.
func sourceUID(rawURL string) (uid, domain string) {
	return sourceUIDDepth(rawURL, 0)
}

func sourceUIDDepth(rawURL string, depth int) (uid, domain string) {
	raw := strings.TrimSpace(rawURL)
	if redirectorHostRe.MatchString(strings.ToLower(raw)) {
		if depth >= 3 {
			return "", ""
		}
		parseable := raw
		if !schemeRe.MatchString(strings.ToLower(parseable)) {
			parseable = "https://" + parseable
		}
		if u, err := url.Parse(parseable); err == nil {
			if dest := strings.TrimSpace(u.Query().Get("u")); dest != "" {
				return sourceUIDDepth(dest, depth+1)
			}
		}
		return "", "" // redirector without a destination carries no identity
	}
	s := normalizeSocial(raw)
	if s == "" {
		return "", ""
	}
	s = socialMirrorPrefixRe.ReplaceAllString(s, "$1")
	if m := fbGroupUIDRe.FindStringSubmatch(s); m != nil && m[1] != "" {
		return "facebook.com/groups/" + m[1], "facebook.com"
	}
	domain = s
	if i := strings.IndexAny(s, "/?"); i >= 0 {
		domain = s[:i]
	}
	return s, domain
}

func uidHash(uid string) string {
	sum := sha256.Sum256([]byte(uid))
	return hex.EncodeToString(sum[:])[:12]
}

// Tracking params that never carry identity on ANY site — stripped when
// cleaning a URL for STORAGE. (normalizeSocial has its own identity-param
// allowlist for social hosts; this list is for conservative cleaning of
// arbitrary websites, where unknown params must be kept.)
var storeTrackingParams = map[string]bool{
	"fbclid": true, "gclid": true, "msclkid": true, "mibextid": true, "igsh": true,
	"ref": true, "refsrc": true, "ref_src": true, "tracking": true,
	"__cft__": true, "__tn__": true,
	"utm_source": true, "utm_medium": true, "utm_campaign": true, "utm_term": true, "utm_content": true,
}

var socialStoreHostRe = regexp.MustCompile(`^(facebook|instagram|tiktok|linkedin|reddit|youtube|twitter|x)\.com(/|\?|$)`)

// canonicalStoreURL cleans a source URL for SAVING (profile
// private_data_sources, collector_config, registry sample_url) — the browsable
// twin of sourceUID's identity key. Operators paste whatever the address bar
// held; every variant of a Facebook group must be stored as exactly
// `https://www.facebook.com/groups/<name>`:
//
//	https://www.facebook.com/groups/examplegroup/          -> https://www.facebook.com/groups/examplegroup
//	https://www.facebook.com/groups/examplegroup?abd=x&y=k -> https://www.facebook.com/groups/examplegroup
//	https://m.facebook.com/groups/examplegroup/about       -> https://www.facebook.com/groups/examplegroup
//
// Known social hosts rebuild from the uid (their vanity paths are
// case-insensitive, so lowercasing is safe; identity params like
// profile.php?id= survive because the uid keeps them). Arbitrary websites are
// cleaned CONSERVATIVELY — path case and unknown params can be load-bearing
// there, so only the fragment, known tracking params, and the trailing slash
// go. ok=false means the URL has no derivable identity (an opaque redirector):
// do not store it, surface it to the operator instead.
func canonicalStoreURL(rawURL string) (clean string, ok bool) {
	raw := strings.TrimSpace(rawURL)
	// Resolve link redirectors to their destination FIRST — the store form of
	// an l.facebook.com wrapper is the wrapped page, never the wrapper.
	for depth := 0; depth < 3 && redirectorHostRe.MatchString(strings.ToLower(raw)); depth++ {
		parseable := raw
		if !schemeRe.MatchString(strings.ToLower(parseable)) {
			parseable = "https://" + parseable
		}
		u, err := url.Parse(parseable)
		if err != nil {
			return "", false
		}
		dest := strings.TrimSpace(u.Query().Get("u"))
		if dest == "" {
			return "", false
		}
		raw = dest
	}
	uid, _ := sourceUID(raw)
	if uid == "" {
		return "", false
	}
	if socialStoreHostRe.MatchString(uid) {
		return "https://www." + uid, true
	}
	parseable := strings.TrimSpace(raw)
	if !schemeRe.MatchString(strings.ToLower(parseable)) {
		parseable = "https://" + parseable
	}
	u, err := url.Parse(parseable)
	if err != nil || u.Host == "" {
		return "https://" + uid, true // uid-shaped fallback beats storing a dirty string
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	q := u.Query()
	for name := range q {
		if storeTrackingParams[strings.ToLower(name)] {
			q.Del(name)
		}
	}
	u.RawQuery = q.Encode()
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), true
}

// --- registry file -------------------------------------------------------------

type registrySubscriber struct {
	ClientSlug   string `json:"client_slug"`
	Priority     string `json:"priority,omitempty"`
	ScanCadence  string `json:"scan_cadence,omitempty"`
	RegisteredAt string `json:"registered_at"`
}

type registryScan struct {
	CompletedAt string `json:"completed_at"`
	RunID       string `json:"run_id"`
	ClientSlug  string `json:"client_slug"`
	DataDir     string `json:"data_dir"`
	Status      string `json:"status"`
	Kind        string `json:"kind,omitempty"` // lane that produced the data: private | public
}

// registryClaim marks a source as being scanned RIGHT NOW, so fifty runs
// firing on the same cron hour do not all scan the same stale source — the
// first `due` claims it, later ones get `wait` until record clears the claim
// or the claim ages past scanClaimTTLHours.
type registryClaim struct {
	ClientSlug string `json:"client_slug"`
	ClaimedAt  string `json:"claimed_at"`
}

type registrySource struct {
	UID         string               `json:"uid"`
	UIDHash     string               `json:"uid_hash"`
	SampleURL   string               `json:"sample_url"`
	Domain      string               `json:"domain"`
	Platform    string               `json:"platform,omitempty"`
	SourceType  string               `json:"source_type,omitempty"`
	Kind        string               `json:"kind,omitempty"` // private | public
	Scope       string               `json:"scope"`          // shared | exclusive
	Subscribers []registrySubscriber `json:"subscribers"`
	LastScan    *registryScan        `json:"last_scan,omitempty"`
	LastFailed  *registryScan        `json:"last_failed,omitempty"`
	ScanClaim   *registryClaim       `json:"scan_claim,omitempty"`
	UpdatedAt   string               `json:"updated_at"`
}

type sourceRegistry struct {
	SchemaVersion     int              `json:"schema_version"`
	FreshnessTTLHours float64          `json:"freshness_ttl_hours"`
	Sources           []registrySource `json:"sources"`
}

func sourceRegistryPath(pipeline string) string {
	return filepath.Join(pipeline, "collector", "source_registry.json")
}

// withLockedJSON runs fn over the JSON document at path under an exclusive
// flock on a sibling .lock file, then atomically rewrites the document.
// Concurrent client runs hit `due`/`record` at the same time; without the
// lock the second writer would silently drop the first one's update.
func withLockedJSON(path string, load func([]byte) error, dump func() ([]byte, error), fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := flockExclusive(lock); err != nil {
		return err
	}
	defer flockUnlock(lock)
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := load(raw); err != nil {
		return err
	}
	if err := fn(); err != nil {
		return err
	}
	out, err := dump()
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func withSourceRegistry(pipeline string, fn func(*sourceRegistry) error) (*sourceRegistry, error) {
	reg := &sourceRegistry{}
	err := withLockedJSON(sourceRegistryPath(pipeline),
		func(raw []byte) error {
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, reg); err != nil {
					return fmt.Errorf("source_registry.json is corrupt: %w", err)
				}
			}
			if reg.SchemaVersion == 0 {
				reg.SchemaVersion = 1
			}
			if reg.FreshnessTTLHours <= 0 {
				reg.FreshnessTTLHours = sourceRegistryDefaultTTLHours
			}
			return nil
		},
		func() ([]byte, error) { return json.MarshalIndent(reg, "", "  ") },
		func() error { return fn(reg) })
	if err != nil {
		return nil, err
	}
	return reg, nil
}

func (r *sourceRegistry) find(uid string) *registrySource {
	for i := range r.Sources {
		if r.Sources[i].UID == uid {
			return &r.Sources[i]
		}
	}
	return nil
}

// upsert finds or creates the entry for rawURL and ensures clientSlug is a
// subscriber. Empty metadata never overwrites existing values.
func (r *sourceRegistry) upsert(rawURL, clientSlug, platform, sourceType, kind, cadence, priority, now string) *registrySource {
	uid, domain := sourceUID(rawURL)
	if uid == "" {
		return nil
	}
	entry := r.find(uid)
	if entry == nil {
		// Auto-created entries (a `due`/`record` on a URL nobody registered)
		// start as "unclassified": they SCAN normally but are never served as
		// reuse until an explicit `register` decides shared vs exclusive —
		// otherwise a client's own asset could silently default to shared.
		r.Sources = append(r.Sources, registrySource{
			UID: uid, UIDHash: uidHash(uid), SampleURL: strings.TrimSpace(rawURL),
			Domain: domain, Scope: "unclassified",
		})
		entry = &r.Sources[len(r.Sources)-1]
	}
	if entry.Platform == "" {
		entry.Platform = platform
	}
	if entry.SourceType == "" {
		entry.SourceType = sourceType
	}
	if entry.Kind == "" {
		entry.Kind = kind
	}
	if clientSlug != "" {
		found := false
		for i := range entry.Subscribers {
			if entry.Subscribers[i].ClientSlug == clientSlug {
				found = true
				if cadence != "" {
					entry.Subscribers[i].ScanCadence = cadence
				}
				if priority != "" {
					entry.Subscribers[i].Priority = priority
				}
			}
		}
		if !found {
			entry.Subscribers = append(entry.Subscribers, registrySubscriber{
				ClientSlug: clientSlug, Priority: priority, ScanCadence: cadence, RegisteredAt: now,
			})
		}
	}
	entry.UpdatedAt = now
	return entry
}

func scanIsFresh(scan *registryScan, ttlHours float64, now time.Time) (bool, float64) {
	if scan == nil || scan.Status != "complete" || scan.CompletedAt == "" {
		return false, 0
	}
	t, err := time.Parse(time.RFC3339, scan.CompletedAt)
	if err != nil {
		return false, 0
	}
	age := now.Sub(t).Hours()
	return age >= 0 && age <= ttlHours, age
}

// --- CLI: tool source-registry -------------------------------------------------

func sourceRegistryUsage() int {
	fmt.Fprintln(os.Stderr, `usage: tool source-registry --pipeline DIR <subcommand>
  register --client S --url U [--platform P --source-type T --kind private|public --cadence C --priority P --scope shared|exclusive]
  due      --client S --kind private|public (--url U ... | --urls-file F)
           -> {"scan":[...],"reuse":[...],"wait":[...]}: scan what is stale, reuse fresh data via pointer,
              wait while another client's claimed scan (< 2h) is still running (use stale_data_dir if present)
  record   --client S --run RUNID --kind private|public (--url U ... | --urls-file F) [--status complete|failed] [--data-dir D]
  set-scope --url U --scope shared|exclusive
  list     [--client S]
  normalize (--url U ... | --urls-file F)   -> clean store-form URLs (no --pipeline needed):
           every pasted Facebook-group variant becomes https://www.facebook.com/groups/<name>;
           save ONLY the returned clean_url into profiles/config`)
	return 2
}

func collectURLArgs(a *cliArgs) ([]string, error) {
	urls := append([]string{}, a.flags["--url"]...)
	if f := a.get("--urls-file"); f != "" {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				urls = append(urls, line)
			}
		}
	}
	return urls, nil
}

func runSourceRegistryCLI(args []string) int {
	valueFlags := map[string]bool{"--pipeline": true, "--client": true, "--url": true,
		"--urls-file": true, "--platform": true, "--source-type": true, "--kind": true,
		"--cadence": true, "--priority": true, "--scope": true, "--run": true,
		"--status": true, "--data-dir": true}
	a, err := parseCLIArgs(args, valueFlags, map[string]bool{})
	if err != nil {
		return crmUsageErr(err.Error())
	}
	if len(a.pos) == 0 {
		return sourceRegistryUsage()
	}
	if a.pos[0] == "normalize" {
		// Pure string cleaning — touches no registry file, needs no --pipeline.
		urls, err := collectURLArgs(a)
		if err != nil {
			return crmFail(err)
		}
		if len(urls) == 0 {
			return crmUsageErr("normalize needs at least one --url (or --urls-file)")
		}
		var out []map[string]any
		for _, u := range urls {
			clean, ok := canonicalStoreURL(u)
			if !ok {
				out = append(out, map[string]any{"url": u, "ok": false,
					"reason": "no_derivable_identity_do_not_store"})
				continue
			}
			uid, _ := sourceUID(u)
			out = append(out, map[string]any{"url": u, "ok": true, "clean_url": clean,
				"changed": clean != strings.TrimSpace(u), "uid": uid, "uid_hash": uidHash(uid)})
		}
		return crmOut(map[string]any{"ok": true, "normalized": out}, 0)
	}
	pipeline := a.get("--pipeline")
	if pipeline == "" {
		return crmUsageErr("--pipeline DIR is required")
	}
	if abs, aerr := filepath.Abs(pipeline); aerr == nil {
		pipeline = abs // registry location and stored data_dir pointers must not depend on the caller's cwd
	}
	cmd := a.pos[0]
	client := a.get("--client")
	kind := strings.ToLower(strings.TrimSpace(a.get("--kind")))
	if kind != "" && kind != "private" && kind != "public" {
		return crmUsageErr("--kind must be private or public")
	}
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	switch cmd {
	case "register":
		urls, err := collectURLArgs(a)
		if err != nil {
			return crmFail(err)
		}
		if client == "" || len(urls) == 0 {
			return crmUsageErr("register needs --client and at least one --url")
		}
		scope := strings.ToLower(strings.TrimSpace(a.get("--scope")))
		if scope != "" && scope != "shared" && scope != "exclusive" {
			return crmUsageErr("register: --scope must be shared or exclusive")
		}
		var made []map[string]any
		_, err = withSourceRegistry(pipeline, func(reg *sourceRegistry) error {
			for _, u := range urls {
				entry := reg.upsert(u, client, a.get("--platform"), a.get("--source-type"),
					kind, a.get("--cadence"), a.get("--priority"), nowStr)
				if entry == nil {
					made = append(made, map[string]any{"url": u, "ok": false, "reason": "unparseable_url"})
					continue
				}
				switch {
				case scope == "shared" && entry.Scope == "exclusive":
					// Never silently widen an exclusive source: that leaks a
					// client-own asset to every subscriber. set-scope is the
					// explicit override.
					made = append(made, map[string]any{"url": u, "ok": false, "uid": entry.UID,
						"reason": "downgrade_refused_use_set_scope", "scope": entry.Scope})
					continue
				case scope != "":
					entry.Scope = scope
				case entry.Scope == "unclassified":
					entry.Scope = "shared" // registration default: third-party sources share
				}
				made = append(made, map[string]any{"url": u, "ok": true, "uid": entry.UID,
					"uid_hash": entry.UIDHash, "scope": entry.Scope})
			}
			return nil
		})
		if err != nil {
			return crmFail(err)
		}
		return crmOut(map[string]any{"ok": true, "registered": made}, 0)

	case "due":
		urls, err := collectURLArgs(a)
		if err != nil {
			return crmFail(err)
		}
		if client == "" || len(urls) == 0 {
			return crmUsageErr("due needs --client and at least one --url (or --urls-file)")
		}
		var scan, reuse, wait, invalid []map[string]any
		var ttl float64
		_, err = withSourceRegistry(pipeline, func(reg *sourceRegistry) error {
			ttl = reg.FreshnessTTLHours
			for _, u := range urls {
				entry := reg.upsert(u, client, "", "", "", "", "", nowStr)
				if entry == nil {
					invalid = append(invalid, map[string]any{"url": u, "reason": "unparseable_url"})
					continue
				}
				wantKind := kind
				if wantKind == "" {
					wantKind = entry.Kind
				}
				fresh, age := scanIsFresh(entry.LastScan, ttl, now)
				kindMatches := wantKind == "" || entry.LastScan == nil || entry.LastScan.Kind == wantKind
				if entry.Scope == "shared" && fresh && kindMatches {
					reuse = append(reuse, map[string]any{
						"url": u, "uid": entry.UID, "uid_hash": entry.UIDHash,
						"data_dir": entry.LastScan.DataDir,
						"completed_at": entry.LastScan.CompletedAt, "scanned_by": entry.LastScan.ClientSlug,
						"run_id": entry.LastScan.RunID, "age_hours": float64(int(age*10)) / 10,
					})
					continue
				}
				// Another client claimed this source within the claim TTL and is
				// scanning right now — wait instead of piling on a duplicate scan.
				if c := entry.ScanClaim; c != nil && c.ClientSlug != client && entry.Scope == "shared" {
					if t, perr := time.Parse(time.RFC3339, c.ClaimedAt); perr == nil {
						if hrs := now.Sub(t).Hours(); hrs >= 0 && hrs <= scanClaimTTLHours {
							w := map[string]any{"url": u, "uid": entry.UID, "uid_hash": entry.UIDHash,
								"claimed_by": c.ClientSlug, "claimed_at": c.ClaimedAt}
							if entry.LastScan != nil && entry.LastScan.Status == "complete" {
								w["stale_data_dir"] = entry.LastScan.DataDir
								w["stale_completed_at"] = entry.LastScan.CompletedAt
							}
							wait = append(wait, w)
							continue
						}
					}
				}
				entry.ScanClaim = &registryClaim{ClientSlug: client, ClaimedAt: nowStr}
				scan = append(scan, map[string]any{"url": u, "uid": entry.UID,
					"uid_hash": entry.UIDHash, "scope": entry.Scope})
			}
			return nil
		})
		if err != nil {
			return crmFail(err)
		}
		return crmOut(map[string]any{"ok": true, "client": client, "ttl_hours": ttl,
			"scan": scan, "reuse": reuse, "wait": wait, "invalid": invalid}, 0)

	case "record":
		urls, err := collectURLArgs(a)
		if err != nil {
			return crmFail(err)
		}
		runID := a.get("--run")
		if client == "" || runID == "" || len(urls) == 0 {
			return crmUsageErr("record needs --client, --run and at least one --url (or --urls-file)")
		}
		status := strings.ToLower(strings.TrimSpace(a.get("--status")))
		if status == "" {
			status = "complete"
		}
		if status != "complete" && status != "failed" {
			return crmUsageErr("record: --status must be complete or failed")
		}
		dataDir := a.get("--data-dir")
		if dataDir == "" {
			dataDir = filepath.Join(pipeline, "collector", "inbox",
				monthForRun(runID, now), safeFilename(client), safeFilename(runID))
		} else if abs, aerr := filepath.Abs(dataDir); aerr == nil {
			dataDir = abs
		}
		if status == "complete" {
			if st, serr := os.Stat(dataDir); serr != nil || !st.IsDir() {
				return crmFail(fmt.Errorf("data_dir does not exist: %s (pass --data-dir with the run's real output dir)", dataDir))
			}
		}
		var recorded []map[string]any
		_, err = withSourceRegistry(pipeline, func(reg *sourceRegistry) error {
			for _, u := range urls {
				entry := reg.upsert(u, client, "", "", kind, "", "", nowStr)
				if entry == nil {
					recorded = append(recorded, map[string]any{"url": u, "ok": false, "reason": "unparseable_url"})
					continue
				}
				scanKind := kind
				if scanKind == "" {
					scanKind = entry.Kind
				}
				scanRec := &registryScan{CompletedAt: nowStr, RunID: runID, ClientSlug: client,
					DataDir: dataDir, Status: status, Kind: scanKind}
				if status == "complete" {
					entry.LastScan = scanRec
				} else {
					entry.LastFailed = scanRec
				}
				if entry.ScanClaim != nil && entry.ScanClaim.ClientSlug == client {
					entry.ScanClaim = nil
				}
				recorded = append(recorded, map[string]any{"url": u, "ok": true, "uid": entry.UID, "status": status})
			}
			return nil
		})
		if err != nil {
			return crmFail(err)
		}
		return crmOut(map[string]any{"ok": true, "recorded": recorded}, 0)

	case "set-scope":
		urls, err := collectURLArgs(a)
		if err != nil {
			return crmFail(err)
		}
		scope := a.get("--scope")
		if len(urls) == 0 || (scope != "shared" && scope != "exclusive") {
			return crmUsageErr("set-scope needs --url and --scope shared|exclusive")
		}
		var changed []map[string]any
		_, err = withSourceRegistry(pipeline, func(reg *sourceRegistry) error {
			for _, u := range urls {
				entry := reg.upsert(u, "", "", "", "", "", "", nowStr)
				if entry == nil {
					changed = append(changed, map[string]any{"url": u, "ok": false, "reason": "unparseable_url"})
					continue
				}
				entry.Scope = scope
				changed = append(changed, map[string]any{"url": u, "ok": true, "uid": entry.UID, "scope": scope})
			}
			return nil
		})
		if err != nil {
			return crmFail(err)
		}
		return crmOut(map[string]any{"ok": true, "changed": changed}, 0)

	case "list":
		var out []registrySource
		_, err := withSourceRegistry(pipeline, func(reg *sourceRegistry) error {
			for _, s := range reg.Sources {
				if client != "" {
					match := false
					for _, sub := range s.Subscribers {
						if sub.ClientSlug == client {
							match = true
						}
					}
					if !match {
						continue
					}
				}
				out = append(out, s)
			}
			sort.Slice(out, func(i, j int) bool { return out[i].UID < out[j].UID })
			return nil
		})
		if err != nil {
			return crmFail(err)
		}
		return crmOut(map[string]any{"ok": true, "count": len(out), "sources": out}, 0)
	}
	return sourceRegistryUsage()
}

// --- CLI: tool search-pool -----------------------------------------------------
//
// Same check-before-work pattern for public keyword searches: same-industry
// clients share one day's results for one normalized keyword instead of each
// re-running the identical web search.

type searchPoolEntry struct {
	Industry    string           `json:"industry"`
	Keyword     string           `json:"keyword"`
	KeywordNorm string           `json:"keyword_norm"`
	SearchedAt  string           `json:"searched_at"`
	ClientSlug  string           `json:"client_slug"`
	Results     []map[string]any `json:"results"`
}

type searchPool struct {
	SchemaVersion     int               `json:"schema_version"`
	FreshnessTTLHours float64           `json:"freshness_ttl_hours"`
	Entries           []searchPoolEntry `json:"entries"`
}

func searchPoolPath(pipeline string) string {
	return filepath.Join(pipeline, "collector", "search_pool.json")
}

var whitespaceRe = regexp.MustCompile(`\s+`)

func normKeyword(s string) string {
	return whitespaceRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), " ")
}

func withSearchPool(pipeline string, fn func(*searchPool) error) error {
	pool := &searchPool{}
	return withLockedJSON(searchPoolPath(pipeline),
		func(raw []byte) error {
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, pool); err != nil {
					return fmt.Errorf("search_pool.json is corrupt: %w", err)
				}
			}
			if pool.SchemaVersion == 0 {
				pool.SchemaVersion = 1
			}
			if pool.FreshnessTTLHours <= 0 {
				pool.FreshnessTTLHours = sourceRegistryDefaultTTLHours
			}
			return nil
		},
		func() ([]byte, error) { return json.MarshalIndent(pool, "", "  ") },
		func() error { return fn(pool) })
}

func runSearchPoolCLI(args []string) int {
	valueFlags := map[string]bool{"--pipeline": true, "--client": true, "--industry": true,
		"--keyword": true, "--results-file": true}
	a, err := parseCLIArgs(args, valueFlags, map[string]bool{})
	if err != nil {
		return crmUsageErr(err.Error())
	}
	usage := func() int {
		fmt.Fprintln(os.Stderr, `usage: tool search-pool --pipeline DIR <subcommand>
  check  --industry I --keyword K                       -> {"fresh":true,"entry":{...}} | {"fresh":false}
  record --client S --industry I --keyword K --results-file F   (F = JSON array of {url,title,note}, or - for stdin)`)
		return 2
	}
	if len(a.pos) == 0 {
		return usage()
	}
	pipeline := a.get("--pipeline")
	if pipeline == "" {
		return crmUsageErr("--pipeline DIR is required")
	}
	industry := normKeyword(a.get("--industry"))
	keyword := normKeyword(a.get("--keyword"))
	now := time.Now().UTC()

	switch a.pos[0] {
	case "check":
		if industry == "" || keyword == "" {
			return crmUsageErr("check needs --industry and --keyword")
		}
		var found *searchPoolEntry
		var ttl float64
		err := withSearchPool(pipeline, func(pool *searchPool) error {
			ttl = pool.FreshnessTTLHours
			for i := range pool.Entries {
				e := &pool.Entries[i]
				if e.Industry == industry && e.KeywordNorm == keyword {
					if t, perr := time.Parse(time.RFC3339, e.SearchedAt); perr == nil {
						if age := now.Sub(t).Hours(); age >= 0 && age <= ttl {
							found = e
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			return crmFail(err)
		}
		if found == nil {
			return crmOut(map[string]any{"ok": true, "fresh": false}, 0)
		}
		return crmOut(map[string]any{"ok": true, "fresh": true, "entry": found}, 0)

	case "record":
		client := a.get("--client")
		resultsFile := a.get("--results-file")
		if client == "" || industry == "" || keyword == "" || resultsFile == "" {
			return crmUsageErr("record needs --client, --industry, --keyword and --results-file")
		}
		var raw []byte
		if resultsFile == "-" {
			raw, err = readAllStdin()
		} else {
			raw, err = os.ReadFile(resultsFile)
		}
		if err != nil {
			return crmFail(err)
		}
		var results []map[string]any
		if err := json.Unmarshal(raw, &results); err != nil {
			return crmFail(fmt.Errorf("results must be a JSON array of objects: %w", err))
		}
		err = withSearchPool(pipeline, func(pool *searchPool) error {
			kept := pool.Entries[:0]
			for _, e := range pool.Entries {
				if e.Industry == industry && e.KeywordNorm == keyword {
					continue // replaced below
				}
				if t, perr := time.Parse(time.RFC3339, e.SearchedAt); perr == nil {
					if now.Sub(t).Hours() > searchPoolPruneDays*24 {
						continue // stale beyond any reuse window; keep the file small
					}
				}
				kept = append(kept, e)
			}
			pool.Entries = append(kept, searchPoolEntry{
				Industry: industry, Keyword: strings.TrimSpace(a.get("--keyword")),
				KeywordNorm: keyword, SearchedAt: now.Format(time.RFC3339),
				ClientSlug: client, Results: results,
			})
			return nil
		})
		if err != nil {
			return crmFail(err)
		}
		return crmOut(map[string]any{"ok": true, "industry": industry, "keyword": keyword,
			"results": len(results)}, 0)
	}
	return usage()
}

func readAllStdin() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}
