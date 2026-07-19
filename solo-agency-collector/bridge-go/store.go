package main

// store.go — Go port of outreach/tools/storage/ (adapter.py + json_adapter.py).
//
// Contract: behave exactly like the Python storage layer on the same files —
// same paths, same lock names, same seq counters, same JSON semantics (key
// order may differ; readers parse, never byte-compare). Records are
// map[string]any so unknown fields survive read-modify-write untouched.
// The fcntl-only restriction is gone: locking is flock on unix and
// LockFileEx on Windows (see flock_*.go) — this is what unlocks Windows.

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// storageError mirrors Python StorageError: CLI prints "StorageError: <msg>" and exits 1.
type storageError struct{ msg string }

func (e *storageError) Error() string { return e.msg }

func storageErrf(format string, a ...any) error {
	return &storageError{msg: fmt.Sprintf(format, a...)}
}

// --- timestamps (injectable clock, DESIGN §17) --------------------------------

func envTruthy(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

var dateOnlyRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// fakeNowOverride honors OUTREACHCRM_FAKE_NOW only under OUTREACHCRM_TEST_MODE,
// exactly like adapter.py. Malformed value in test mode fails loudly.
func fakeNowOverride() (string, error) {
	if !envTruthy("OUTREACHCRM_TEST_MODE") {
		return "", nil
	}
	raw := strings.TrimSpace(os.Getenv("OUTREACHCRM_FAKE_NOW"))
	if raw == "" {
		return "", nil
	}
	s := raw
	if dateOnlyRe.MatchString(raw) {
		s = raw + "T00:00:00Z"
	}
	t, err := parseISO(s)
	if err != nil {
		return "", storageErrf("OUTREACHCRM_FAKE_NOW is not a valid ISO-8601 timestamp: %q (expected YYYY-MM-DD or YYYY-MM-DDTHH:MM:SSZ)", raw)
	}
	return t.UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z"), nil
}

// parseISO accepts the shapes Python's fromisoformat handles in this codebase:
// with/without offset, trailing Z.
func parseISO(s string) (time.Time, error) {
	s = strings.Replace(s, "Z", "+00:00", 1)
	for _, layout := range []string{"2006-01-02T15:04:05-07:00", "2006-01-02T15:04:05.999999-07:00", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("bad iso timestamp %q", s)
}

func nowISO() string {
	if fake, err := fakeNowOverride(); err != nil {
		panic(err) // matches Python raising loudly in test mode
	} else if fake != "" {
		return fake
	}
	return time.Now().UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z")
}

func todayStr(now string) string {
	if now == "" {
		now = nowISO()
	}
	return now[:10]
}

func monthStr(now string) string {
	if now == "" {
		now = nowISO()
	}
	return now[:7]
}

// isoDaysAgoFrom returns the ISO timestamp `days` before the given now — exact
// 24h days like Python timedelta, not calendar arithmetic.
func isoDaysAgoFrom(days int, nowStr string) string {
	t, err := parseISO(nowStr)
	if err != nil {
		t = time.Now().UTC()
	}
	return t.UTC().Add(-time.Duration(days) * 24 * time.Hour).Truncate(time.Second).Format("2006-01-02T15:04:05Z")
}

func isoDaysAgo(days int) string { return isoDaysAgoFrom(days, nowISO()) }

// --- ULID (Crockford base32, 48-bit time + 80-bit randomness) -----------------

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

func newULID(prefix string) string {
	ms := uint64(time.Now().UnixMilli()) & ((1 << 48) - 1)
	var rb [10]byte
	if _, err := rand.Read(rb[:]); err != nil {
		panic(err)
	}
	// value = ms<<80 | rand as a 128-bit big-endian quantity; emit 26 chars LSB-first reversed.
	var buf [16]byte // 128 bits
	for i := 0; i < 6; i++ {
		buf[i] = byte(ms >> (8 * (5 - i)))
	}
	copy(buf[6:], rb[:])
	// read 5 bits at a time from the low end
	chars := make([]byte, 26)
	// convert buf (big-endian 128-bit) into bits; extract 26 5-bit groups from LSB
	for i := 0; i < 26; i++ {
		bitPos := i * 5
		v := 0
		for b := 0; b < 5; b++ {
			byteIdx := 15 - (bitPos+b)/8
			bitIdx := (bitPos + b) % 8
			if byteIdx >= 0 && buf[byteIdx]&(1<<bitIdx) != 0 {
				v |= 1 << b
			}
		}
		chars[25-i] = crockford[v]
	}
	return prefix + string(chars)
}

// --- identity normalization ---------------------------------------------------

func normalizeEmail(address string) string {
	return strings.ToLower(strings.TrimSpace(address))
}

var nonDigitRe = regexp.MustCompile(`\D`)

func normalizePhone(number string) string {
	if number == "" {
		return ""
	}
	s := number
	if i := strings.Index(s, "x"); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, "ext"); i >= 0 {
		s = s[:i]
	}
	plus := strings.HasPrefix(strings.TrimSpace(s), "+")
	digits := nonDigitRe.ReplaceAllString(s, "")
	if digits == "" {
		return ""
	}
	if len(digits) < 10 || len(digits) > 15 {
		return ""
	}
	uniq := map[byte]bool{}
	for i := 0; i < len(digits); i++ {
		uniq[digits[i]] = true
	}
	if len(uniq) == 1 {
		return ""
	}
	if plus {
		return "+" + digits
	}
	if len(digits) == 10 {
		return "+1" + digits
	}
	if len(digits) == 11 && strings.HasPrefix(digits, "1") {
		return "+" + digits
	}
	return "+" + digits
}

var schemeRe = regexp.MustCompile(`^https?://`)
var wwwRe = regexp.MustCompile(`^www\.`)

func normalizeSocial(url string) string {
	if url == "" {
		return ""
	}
	s := strings.ToLower(strings.TrimSpace(url))
	s = schemeRe.ReplaceAllString(s, "")
	s = wwwRe.ReplaceAllString(s, "")
	if i := strings.Index(s, "?"); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, "#"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, "/")
}

// --- Cond matching (flat fields, dotted paths) --------------------------------

type cond struct {
	Field string
	Op    string
	Value any
}

var allowedOps = map[string]bool{"=": true, "!=": true, "<": true, ">": true, "contains": true, "in": true}

func digGet(record map[string]any, dotted string) any {
	var cur any = record
	for _, part := range strings.Split(dotted, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		v, ok := m[part]
		if !ok {
			return nil
		}
		cur = v
	}
	return cur
}

// jsonEq mirrors Python ==: numbers compare numerically, strings/bools by value.
func jsonEq(a, b any) bool {
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if aok && bok {
		return af == bf
	}
	return a == b
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	}
	return 0, false
}

// jsonLess mirrors Python < with TypeError->no-match: only same-kind compares succeed.
func jsonLess(a, b any) (bool, bool) {
	if af, aok := toFloat(a); aok {
		if bf, bok := toFloat(b); bok {
			return af < bf, true
		}
		return false, false
	}
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return as < bs, true
	}
	return false, false
}

func condMatches(record map[string]any, where []cond) (bool, error) {
	for _, c := range where {
		if !allowedOps[c.Op] {
			return false, storageErrf("unsupported op %q", c.Op)
		}
		actual := digGet(record, c.Field)
		switch c.Op {
		case "=":
			if !jsonEq(actual, c.Value) {
				return false, nil
			}
		case "!=":
			if jsonEq(actual, c.Value) {
				return false, nil
			}
		case "<":
			ok, cmp := jsonLess(actual, c.Value)
			if actual == nil || !cmp || !ok {
				return false, nil
			}
		case ">":
			ok, cmp := jsonLess(c.Value, actual)
			if actual == nil || !cmp || !ok {
				return false, nil
			}
		case "contains":
			if !condContains(actual, c.Value) {
				return false, nil
			}
		case "in":
			list, ok := c.Value.([]any)
			if !ok {
				return false, nil // non-iterable (or string) value -> never a match
			}
			found := false
			for _, item := range list {
				if jsonEq(actual, item) {
					found = true
					break
				}
			}
			if !found {
				return false, nil
			}
		}
	}
	return true, nil
}

func condContains(actual, value any) bool {
	switch a := actual.(type) {
	case string:
		s, ok := value.(string)
		return ok && strings.Contains(a, s)
	case []any:
		for _, item := range a {
			if jsonEq(item, value) {
				return true
			}
		}
	}
	return false
}

// --- JSON helpers (Python-compatible output semantics) ------------------------

// marshalIndentJSON = json.dumps(obj, ensure_ascii=False, indent=2): 2-space
// indent, no HTML escaping, no trailing newline. Key order is Go-sorted (differs
// from Python insertion order; all readers parse, none byte-compare).
func marshalIndentJSON(v any) string {
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		panic(err)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// marshalLineJSON = one compact JSONL line (no trailing newline).
func marshalLineJSON(v any) string {
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(err)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func readJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func readJSONLines(path string) []map[string]any {
	var out []map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

// --- map accessors (the price of map[string]any records) ----------------------

func mGet(m map[string]any, key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

func mStr(m map[string]any, key string) string {
	s, _ := mGet(m, key).(string)
	return s
}

func mMap(m map[string]any, key string) map[string]any {
	mm, _ := mGet(m, key).(map[string]any)
	return mm
}

func mList(m map[string]any, key string) []any {
	l, _ := mGet(m, key).([]any)
	return l
}

func mBool(m map[string]any, key string) bool {
	b, _ := mGet(m, key).(bool)
	return b
}

func mInt(m map[string]any, key string, def int) int {
	if f, ok := toFloat(mGet(m, key)); ok {
		return int(f)
	}
	if s, ok := mGet(m, key).(string); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			return n
		}
	}
	return def
}

// asFloat mirrors crm_store._as_float: never raises, defaults on junk.
func asFloat(v any, def float64) float64 {
	if v == nil {
		return def
	}
	if f, ok := toFloat(v); ok {
		return f
	}
	if s, ok := v.(string); ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return f
		}
	}
	return def
}

func mapsOf(l []any) []map[string]any {
	var out []map[string]any
	for _, v := range l {
		if m, ok := v.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// --- JSON store (json_adapter.py port) ----------------------------------------

var logPaths = map[string]string{"tasks": "tasks/tasks.jsonl"}
var reservedCollections = map[string]bool{"activities": true, "tasks": true, "contact_identities": true, "suppression": true}

type jsonStore struct {
	clientRoot string
	crmRoot    string
	locksDir   string

	identityCache    map[[2]string]string // (kind,value) -> contact_id ("" = removed)
	identityCacheSig [2]int64
	identityCacheSet bool
}

func newJSONStore(clientRoot string) *jsonStore {
	abs, err := filepath.Abs(clientRoot)
	if err != nil {
		abs = clientRoot
	}
	return &jsonStore{
		clientRoot: abs,
		crmRoot:    filepath.Join(abs, "crm"),
		locksDir:   filepath.Join(abs, "crm", ".locks"),
	}
}

func safeID(id string) error {
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "\\") || id == "." || id == ".." || strings.HasPrefix(id, ".") {
		return storageErrf("unsafe id %q", id)
	}
	return nil
}

func (s *jsonStore) collectionDir(collection string) (string, error) {
	if reservedCollections[collection] {
		return "", storageErrf("%q is a log, not a record collection", collection)
	}
	if err := safeID(collection); err != nil {
		return "", err
	}
	// safeID above already rejects separators and dot-names; this lexical check
	// mirrors Python's realpath/commonpath guard without tripping on macOS's
	// /var -> /private/var symlink (both sides stay unresolved).
	d := filepath.Clean(filepath.Join(s.crmRoot, collection))
	rootSep := filepath.Clean(s.crmRoot) + string(filepath.Separator)
	if !strings.HasPrefix(d, rootSep) {
		return "", storageErrf("collection %q escapes the client crm root", collection)
	}
	return d, nil
}

func (s *jsonStore) recordPath(collection, id string) (string, error) {
	if err := safeID(id); err != nil {
		return "", err
	}
	d, err := s.collectionDir(collection)
	if err != nil {
		return "", err
	}
	return filepath.Join(d, id+".json"), nil
}

func (s *jsonStore) logPath(log, when string) (string, error) {
	if log == "activities" {
		return filepath.Join(s.crmRoot, "activities", monthStr(when), "activities.jsonl"), nil
	}
	if p, ok := logPaths[log]; ok {
		return filepath.Join(s.crmRoot, p), nil
	}
	if err := safeID(log); err != nil {
		return "", err
	}
	return filepath.Join(s.crmRoot, log+".jsonl"), nil
}

// lock takes the named exclusive lock; the returned func releases it.
func (s *jsonStore) lock(name string) (func(), error) {
	if err := os.MkdirAll(s.locksDir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(s.locksDir, name+".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	if err := flockExclusive(f); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		flockUnlock(f)
		f.Close()
	}, nil
}

// atomicWriteFile = temp (unique per call) + fsync + rename.
func atomicWriteFile(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var rb [8]byte
	_, _ = rand.Read(rb[:])
	tmp := fmt.Sprintf("%s.tmp.%d.%x", path, os.Getpid(), rb)
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func (s *jsonStore) nextSeq(log string) (int, error) {
	seqFile := filepath.Join(s.crmRoot, ".seq", strings.ReplaceAll(log, "/", "_")+".seq")
	if err := os.MkdirAll(filepath.Dir(seqFile), 0o755); err != nil {
		return 0, err
	}
	cur := 0
	if data, err := os.ReadFile(seqFile); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			cur = n
		}
	}
	nxt := cur + 1
	if err := atomicWriteFile(seqFile, strconv.Itoa(nxt)); err != nil {
		return 0, err
	}
	return nxt, nil
}

// --- records ---

func (s *jsonStore) get(collection, id string) (map[string]any, error) {
	path, err := s.recordPath(collection, id)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}
	return readJSONFile(path)
}

func (s *jsonStore) put(collection, id string, record map[string]any) error {
	rec := make(map[string]any, len(record)+4)
	for k, v := range record {
		rec[k] = v
	}
	if _, ok := rec["id"]; !ok {
		rec["id"] = id
	}
	if rec["id"] != id {
		return storageErrf("record id %v != path id %q", rec["id"], id)
	}
	if _, ok := rec["schema_version"]; !ok {
		rec["schema_version"] = 1
	}
	if _, ok := rec["created_at"]; !ok {
		rec["created_at"] = nowISO()
	}
	rec["updated_at"] = nowISO()
	unlock, err := s.lock("col_" + collection)
	if err != nil {
		return err
	}
	defer unlock()
	path, err := s.recordPath(collection, id)
	if err != nil {
		return err
	}
	return atomicWriteFile(path, marshalIndentJSON(rec))
}

func (s *jsonStore) update(collection, id string, mutate func(map[string]any) map[string]any) (map[string]any, error) {
	unlock, err := s.lock("col_" + collection)
	if err != nil {
		return nil, err
	}
	defer unlock()
	path, err := s.recordPath(collection, id)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, storageErrf("%s/%s not found for update", collection, id)
	}
	rec, err := readJSONFile(path)
	if err != nil {
		return nil, err
	}
	rec = mutate(rec)
	rec["updated_at"] = nowISO()
	if err := atomicWriteFile(path, marshalIndentJSON(rec)); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *jsonStore) delete(collection, id string) error {
	unlock, err := s.lock("col_" + collection)
	if err != nil {
		return err
	}
	defer unlock()
	path, err := s.recordPath(collection, id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return os.Remove(path)
	}
	return nil
}

type sortSpec struct {
	Field string
	Desc  bool
}

func (s *jsonStore) query(collection string, where []cond, sortBy *sortSpec, limit, offset int) ([]map[string]any, error) {
	d, err := s.collectionDir(collection)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	entries, _ := os.ReadDir(d)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		rec, err := readJSONFile(filepath.Join(d, name))
		if err != nil {
			continue
		}
		ok, err := condMatches(rec, where)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, rec)
		}
	}
	if sortBy != nil {
		field, desc := sortBy.Field, sortBy.Desc
		sort.SliceStable(out, func(i, j int) bool {
			a, b := out[i][field], out[j][field]
			// Python key: (value is None, value); reverse on desc
			less := func(x, y any) bool {
				if (x == nil) != (y == nil) {
					return y == nil // non-nil sorts before nil
				}
				if x == nil {
					return false
				}
				if l, ok := jsonLess(x, y); ok {
					return l
				}
				return false
			}
			if desc {
				return less(b, a)
			}
			return less(a, b)
		})
	}
	if offset > 0 {
		if offset >= len(out) {
			out = nil
		} else {
			out = out[offset:]
		}
	}
	if limit >= 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

// --- append-only logs ---

func (s *jsonStore) appendLog(log string, record map[string]any) (map[string]any, error) {
	unlock, err := s.lock("log_" + log)
	if err != nil {
		return nil, err
	}
	defer unlock()
	rec := make(map[string]any, len(record)+2)
	for k, v := range record {
		rec[k] = v
	}
	seq, err := s.nextSeq(log)
	if err != nil {
		return nil, err
	}
	rec["seq"] = seq
	if _, ok := rec["ts"]; !ok {
		rec["ts"] = nowISO()
	}
	path, err := s.logPath(log, mStr(rec, "ts"))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.WriteString(marshalLineJSON(rec) + "\n"); err != nil {
		return nil, err
	}
	if err := f.Sync(); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *jsonStore) readLog(log string, sinceSeq int, where []cond) ([]map[string]any, error) {
	var files []string
	if log == "activities" {
		base := filepath.Join(s.crmRoot, "activities")
		entries, _ := os.ReadDir(base)
		months := make([]string, 0, len(entries))
		for _, e := range entries {
			months = append(months, e.Name())
		}
		sort.Strings(months)
		for _, m := range months {
			p := filepath.Join(base, m, "activities.jsonl")
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				files = append(files, p)
			}
		}
	} else {
		p, err := s.logPath(log, "")
		if err != nil {
			return nil, err
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			files = append(files, p)
		}
	}
	var rows []map[string]any
	for _, p := range files {
		for _, rec := range readJSONLines(p) {
			if sinceSeq >= 0 {
				if seq, ok := toFloat(rec["seq"]); ok && int(seq) <= sinceSeq {
					continue
				} else if !ok && 0 <= sinceSeq {
					continue
				}
			}
			ok, err := condMatches(rec, where)
			if err != nil {
				return nil, err
			}
			if ok {
				rows = append(rows, rec)
			}
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, _ := toFloat(rows[i]["seq"])
		b, _ := toFloat(rows[j]["seq"])
		return a < b
	})
	return rows, nil
}

// --- identity reverse index ---

func (s *jsonStore) identityLogSig() [2]int64 {
	p, err := s.logPath("contact_identities", "")
	if err != nil {
		return [2]int64{0, 0}
	}
	st, err := os.Stat(p)
	if err != nil {
		return [2]int64{0, 0}
	}
	return [2]int64{st.Size(), st.ModTime().UnixNano()}
}

func (s *jsonStore) ensureIdentityCache() map[[2]string]string {
	sig := s.identityLogSig()
	if !s.identityCacheSet || sig != s.identityCacheSig {
		cache := map[[2]string]string{}
		rows, _ := s.readLog("contact_identities", -1, nil)
		for _, rec := range rows {
			key := [2]string{mStr(rec, "kind"), mStr(rec, "value")}
			if mBool(rec, "removed") {
				cache[key] = ""
			} else {
				cache[key] = mStr(rec, "contact_id")
			}
		}
		s.identityCache = cache
		s.identityCacheSig = sig
		s.identityCacheSet = true
	}
	return s.identityCache
}

func (s *jsonStore) registerIdentity(kind, normalizedValue, contactID string) error {
	if normalizedValue == "" {
		return nil
	}
	_, err := s.appendLog("contact_identities", map[string]any{
		"kind": kind, "value": normalizedValue, "contact_id": contactID, "removed": false,
	})
	if err != nil {
		return err
	}
	if s.identityCacheSet {
		s.identityCache[[2]string{kind, normalizedValue}] = contactID
		s.identityCacheSig = s.identityLogSig()
	}
	return nil
}

func (s *jsonStore) findByIdentity(kind, normalizedValue string) string {
	if normalizedValue == "" {
		return ""
	}
	return s.ensureIdentityCache()[[2]string{kind, normalizedValue}]
}

// --- atomic quota reservation ---

func (s *jsonStore) reservationPath(sendboxSlug, day string) (string, error) {
	if err := safeID(sendboxSlug); err != nil {
		return "", err
	}
	return filepath.Join(s.clientRoot, "sendboxes", "_reservations", sendboxSlug, day+".json"), nil
}

func loadReservationState(path string) map[string]any {
	if st, err := readJSONFile(path); err == nil && st != nil {
		if _, ok := st["count"]; !ok {
			st["count"] = 0
		}
		if _, ok := st["tokens"]; !ok {
			st["tokens"] = []any{}
		}
		return st
	}
	return map[string]any{"count": 0, "tokens": []any{}}
}

func (s *jsonStore) reserve(sendboxSlug, day string, cap int) (string, error) {
	path, err := s.reservationPath(sendboxSlug, day)
	if err != nil {
		return "", err
	}
	unlock, err := s.lock("reserve_" + sendboxSlug + "_" + day)
	if err != nil {
		return "", err
	}
	defer unlock()
	state := loadReservationState(path)
	if mInt(state, "count", 0) >= cap {
		return "", nil
	}
	token := newULID("rsv_")
	state["count"] = mInt(state, "count", 0) + 1
	state["tokens"] = append(mList(state, "tokens"), map[string]any{"token": token, "at": nowISO()})
	if err := atomicWriteFile(path, marshalIndentJSON(state)); err != nil {
		return "", err
	}
	return token, nil
}

func (s *jsonStore) release(sendboxSlug, day, token string) (bool, error) {
	path, err := s.reservationPath(sendboxSlug, day)
	if err != nil {
		return false, err
	}
	unlock, err := s.lock("reserve_" + sendboxSlug + "_" + day)
	if err != nil {
		return false, err
	}
	defer unlock()
	state := loadReservationState(path)
	toks := mList(state, "tokens")
	kept := make([]any, 0, len(toks))
	for _, t := range toks {
		if tm, ok := t.(map[string]any); ok && mStr(tm, "token") == token {
			continue
		}
		kept = append(kept, t)
	}
	if len(kept) == len(toks) {
		return false, nil
	}
	state["tokens"] = kept
	c := mInt(state, "count", 0) - 1
	if c < 0 {
		c = 0
	}
	state["count"] = c
	if err := atomicWriteFile(path, marshalIndentJSON(state)); err != nil {
		return false, err
	}
	return true, nil
}

func (s *jsonStore) reservationCount(sendboxSlug, day string) int {
	path, err := s.reservationPath(sendboxSlug, day)
	if err != nil {
		return 0
	}
	return mInt(loadReservationState(path), "count", 0)
}
