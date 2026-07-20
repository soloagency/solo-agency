package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// tracking.go — T2 bridge side: open/click/unsub tracking through WideCast.
//
// The bridge SIGNS tracking links locally (holds the same secret WideCast has),
// embeds them when composing a send, and PULLS the resulting events back into
// outreach/tracking/events.jsonl — where the Sent view already reads them. It
// never depends on WideCast being up to send: a missing/disabled tracking
// config makes every helper a no-op, so sends and the reply poller are
// unaffected. Opens/clicks stay directional-only (doctrine: only a reply drives
// action); the value they add is visibility on the Sent page.
//
// Server contract: docs/WIDECAST_TRACKING_SPEC.md (transcoder/dashboard2.py).

// trackCfg is the per-client tracking configuration, read from the SAME
// provider_config.local.json that holds the WideCast API key. Absent or
// disabled => tracking is off and every entry point below no-ops.
type trackCfg struct {
	Enabled   bool
	BaseURL   string // e.g. https://widecast.ai/app/dashboard  (no trailing slash)
	Company   string // WideCast company_id -> the token tenant `t`
	Secret    string // shared with the server's WIDECAST_WEBHOOK_SECRET
	apiKey    string // wc_live_ key for pulling /v1/track/events
	clientDir string
}

// loadTrackCfg reads the "tracking" block out of the client's provider config.
// Shape:
//
//	{"tracking": {"enabled": true, "base_url": "https://widecast.ai/app/dashboard",
//	              "company_id": "...", "secret": "<same as server>"}}
//
// The wc_live_ key is reused from the active provider block (no second secret).
func loadTrackCfg(clientDir string) trackCfg {
	cfgPath := filepath.Join(clientDir, "..", "integrations", "providers", "provider_config.local.json")
	config, err := providerReadJSON(cfgPath)
	if err != nil {
		return trackCfg{}
	}
	t := mMap(config, "tracking")
	if t == nil || !mBool(t, "enabled") {
		return trackCfg{}
	}
	base := normalizeServerURL(mStr(t, "base_url"))
	secret := mStr(t, "secret")
	company := mStr(t, "company_id")
	if base == "" || secret == "" || company == "" {
		return trackCfg{} // half-configured is off, not a partial hazard
	}
	apiKey, _ := providerAPIKey(config, activeProvider(config, "widecast"))
	return trackCfg{Enabled: true, BaseURL: base, Company: company, Secret: secret,
		apiKey: apiKey, clientDir: clientDir}
}

// trackToken builds the stateless signed token, byte-for-byte compatible with
// the Python packer in dashboard2.py: compact JSON in key order k,t,m[,u]
// (struct field order), HTML-escaping OFF, base64url without padding, then
// ".<HMAC-SHA256(secret,payload)[:16 bytes = 32 hex]>".
func (c trackCfg) trackToken(kind, msgRef, targetURL string) string {
	// struct field order == Python dict insertion order (k,t,m,u); U omitted
	// unless click, matching `if kind == "c"`.
	type tok struct {
		K string `json:"k"`
		T string `json:"t"`
		M string `json:"m"`
		U string `json:"u,omitempty"`
	}
	v := tok{K: kind, T: c.Company, M: msgRef}
	if kind == "c" {
		v.U = targetURL
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	raw := bytes.TrimRight(buf.Bytes(), "\n") // Encoder appends a newline
	payload := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, []byte(c.Secret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))[:32]
	return payload + "." + sig
}

func (c trackCfg) openURL(msgRef string) string {
	return c.BaseURL + "/t/o/" + c.trackToken("o", msgRef, "") + ".gif"
}
func (c trackCfg) clickURL(msgRef, target string) string {
	return c.BaseURL + "/t/c/" + c.trackToken("c", msgRef, target)
}
func (c trackCfg) unsubURL(msgRef string) string {
	return c.BaseURL + "/t/u/" + c.trackToken("u", msgRef, "")
}

// trackHTMLBody rewrites the companion link to a click-tracked URL and appends
// a 1x1 open pixel. Returns the body unchanged if tracking is off. Plain-text
// sends have no HTML part, so they get reply + one-click-unsub only (by design).
func (c trackCfg) trackHTMLBody(htmlBody, msgRef, companionURL string) string {
	if !c.Enabled || htmlBody == "" {
		return htmlBody
	}
	if companionURL != "" && (strings.HasPrefix(companionURL, "http://") || strings.HasPrefix(companionURL, "https://")) {
		// rewrite href="companion" -> href="clickURL". Replace the raw URL
		// wherever it appears in an href; leave visible link text intact.
		click := c.clickURL(msgRef, companionURL)
		htmlBody = strings.ReplaceAll(htmlBody, `href="`+companionURL+`"`, `href="`+click+`"`)
		htmlBody = strings.ReplaceAll(htmlBody, `href='`+companionURL+`'`, `href='`+click+`'`)
	}
	pixel := `<img src="` + c.openURL(msgRef) + `" width="1" height="1" alt="" style="display:none;border:0" />`
	return htmlBody + "\n" + pixel
}

// unsubHeaders returns the one-click List-Unsubscribe pair (RFC 8058) to MERGE
// with the existing mailto header. Empty when tracking is off.
func (c trackCfg) unsubHeader(msgRef, existingMailto string) (listUnsub string, postHeader string) {
	if !c.Enabled {
		return existingMailto, ""
	}
	https := "<" + c.unsubURL(msgRef) + ">"
	if existingMailto != "" {
		return https + ", " + existingMailto, "List-Unsubscribe=One-Click"
	}
	return https, "List-Unsubscribe=One-Click"
}

// ---------- event pull ----------

const trackEventsFile = "tracking/events.jsonl"
const trackCursorFile = "tracking/.pull_cursor.json"

// pollTrackingEvents pulls new open/click/unsub events for this client and
// appends them (bot-tagged) to outreach/tracking/events.jsonl, which the Sent
// view reads. Cursor + boot_id are persisted so a WideCast restart is handled
// (boot changes -> reset cursor, dedup by seq we already stored). Best-effort:
// any error just logs and returns; the poll never blocks the reply poller.
func (c trackCfg) pollTrackingEvents() {
	if !c.Enabled || c.apiKey == "" {
		return
	}
	curPath := filepath.Join(c.clientDir, filepath.FromSlash(trackCursorFile))
	cur, _ := readJSONFile(curPath)
	if cur == nil {
		cur = map[string]any{}
	}
	since := mInt(cur, "cursor", 0)
	boot := mStr(cur, "boot_id")

	u := fmt.Sprintf("%s/v1/track/events?since=%d&boot=%s", c.BaseURL, since, boot)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		log.Printf("tracking: pull %s: %v", filepath.Base(filepath.Dir(c.clientDir)), err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("tracking: pull HTTP %d", resp.StatusCode)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	var pr struct {
		BootID    string           `json:"boot_id"`
		Restarted bool             `json:"restarted"`
		Cursor    int              `json:"cursor"`
		Events    []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(body, &pr); err != nil {
		log.Printf("tracking: bad pull payload: %v", err)
		return
	}
	if len(pr.Events) == 0 {
		// still record the (possibly new) boot_id + cursor so a restart with no
		// events doesn't leave us re-requesting a dead boot forever
		cur["boot_id"], cur["cursor"] = pr.BootID, pr.Cursor
		_ = atomicWriteFile(curPath, marshalIndentJSON(cur))
		return
	}

	// dedup vs what we already wrote (survives a restart-driven replay): key by
	// (m|kind|ts) since seq resets on the server side after a restart
	evPath := filepath.Join(c.clientDir, filepath.FromSlash(trackEventsFile))
	seen := map[string]bool{}
	if pr.Restarted {
		for _, e := range readJSONLines(evPath) {
			seen[evKey(e)] = true
		}
	}
	botHashes := detectBotHashes(pr.Events)
	if err := os.MkdirAll(filepath.Dir(evPath), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(evPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	written := 0
	for _, e := range pr.Events {
		if pr.Restarted && seen[evKey(e)] {
			continue
		}
		if ih := mStr(e, "ip_hash"); ih != "" && botHashes[ih] {
			e["bot"] = true // scanner fan-out; Sent view ignores bot-tagged hits
		}
		e["pulled_at"] = nowISO()
		if _, werr := f.WriteString(marshalLineJSON(e) + "\n"); werr != nil {
			break
		}
		written++
	}
	f.Close()
	cur["boot_id"], cur["cursor"] = pr.BootID, pr.Cursor
	_ = atomicWriteFile(curPath, marshalIndentJSON(cur))
	if written > 0 {
		log.Printf("tracking: %s +%d events", filepath.Base(filepath.Dir(c.clientDir)), written)
	}
	// A one-click unsub is a compliance action, not just a stat: turn it into a
	// suppression + sequence freeze, mirroring the mailto-unsub path in sync.
	c.applyUnsubEvents(pr.Events)
}

// applyUnsubEvents maps unsub events (msg ref -> lead via sent_log) to a real
// suppression + freeze. Idempotent (suppression is a set), so a replayed event
// is harmless. Bot-tagged hits are ignored.
func (c trackCfg) applyUnsubEvents(events []map[string]any) {
	var refs []string
	for _, e := range events {
		if mStr(e, "kind") == "unsub" && !mBool(e, "bot") {
			refs = append(refs, mStr(e, "m"))
		}
	}
	if len(refs) == 0 {
		return
	}
	store := newCrmStore(c.clientDir)
	leadByRef := c.msgRefToLead(store, refs)
	for _, ref := range refs {
		lead := leadByRef[ref]
		if lead == "" {
			continue
		}
		if _, err := store.suppressContact(lead, "unsubscribe", "", "rule"); err != nil {
			log.Printf("tracking: suppress %s: %v", lead, err)
			continue
		}
		_, _ = store.setContact(lead, map[string]any{"sequence_state": "frozen"})
		_, _ = store.logActivity("unsubscribe", lead, "unsubscribed via one-click (tracking)", "rule", nil, nil)
	}
}

// msgRefToLead resolves rfc_message_id -> resolved lead id by scanning sent
// logs once. Only unsub volume (~1%) hits this path.
func (c trackCfg) msgRefToLead(store *crmStore, refs []string) map[string]string {
	want := map[string]bool{}
	for _, r := range refs {
		want[r] = true
	}
	out := map[string]string{}
	for _, p := range store.allSentLogs("") {
		for _, row := range readJSONLines(p) {
			if rid := mStr(row, "rfc_message_id"); want[rid] {
				out[rid] = store.resolve(mStr(row, "lead_id"))
			}
		}
	}
	return out
}

func evKey(e map[string]any) string {
	return mStr(e, "m") + "|" + mStr(e, "kind") + "|" + fmt.Sprint(e["ts"])
}

// detectBotHashes flags ip_hashes that hit many DISTINCT messages in one pull —
// the signature of a security scanner (Outlook SafeLinks etc.) fanning out
// across every link. A human touches one or two of their own messages.
func detectBotHashes(events []map[string]any) map[string]bool {
	msgsByIP := map[string]map[string]bool{}
	for _, e := range events {
		ih := mStr(e, "ip_hash")
		if ih == "" {
			continue
		}
		if msgsByIP[ih] == nil {
			msgsByIP[ih] = map[string]bool{}
		}
		msgsByIP[ih][mStr(e, "m")] = true
	}
	bots := map[string]bool{}
	for ih, msgs := range msgsByIP {
		if len(msgs) >= 5 { // one IP, 5+ different emails in a single window = scanner
			bots[ih] = true
		}
	}
	return bots
}

// pollTrackingForClients is called from the reply-poller tick for every client.
func (b *bridge) pollTrackingForClients() {
	for _, c := range b.uiClients() {
		cfg := loadTrackCfg(filepath.Join(c.Path, "outreach"))
		if cfg.Enabled {
			cfg.pollTrackingEvents()
		}
	}
}
