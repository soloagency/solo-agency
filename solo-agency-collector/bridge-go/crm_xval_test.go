package main

// crm_xval_test.go — golden cross-validation of the Go crm-store port against
// the Python original (outreach/tools/crm_store.py), per docs/UI_DESIGN.md §8:
// the same CLI scenario runs against two twin workspaces (one per
// implementation) under the injected test clock; every command's stdout and the
// final file trees must be semantically identical after canonicalization
// (generated ULIDs -> stable placeholders, absolute roots stripped, key order
// ignored, HTML render artifacts excluded — the Python side shells a renderer
// the Go side inlines later).
//
// Skips when python3 or the repo checkout isn't available (e.g. in a bare
// binary distribution).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func findCrmStorePy(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "..", "outreach", "tools", "crm_store.py"))
	if err != nil {
		t.Skip("cannot resolve repo path")
	}
	if _, err := os.Stat(p); err != nil {
		t.Skip("crm_store.py not present (not a repo checkout)")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	return p
}

type xstep struct {
	FakeNow   string
	Argv      []string
	WantFail  string // non-empty => expect nonzero exit and this substring on stderr (both sides)
	IgnoreKey []string
}

type xresult struct {
	Code   int
	Stdout string
	Stderr string
}

func runPyStep(t *testing.T, pyTool string, s xstep) xresult {
	t.Helper()
	cmd := exec.Command("python3", append([]string{pyTool}, s.Argv...)...)
	cmd.Env = append(os.Environ(), "OUTREACHCRM_TEST_MODE=1", "OUTREACHCRM_FAKE_NOW="+s.FakeNow)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("python step %v: %v", s.Argv, err)
	}
	return xresult{code, out.String(), errb.String()}
}

func runGoStep(t *testing.T, s xstep) xresult {
	t.Helper()
	t.Setenv("OUTREACHCRM_TEST_MODE", "1")
	t.Setenv("OUTREACHCRM_FAKE_NOW", s.FakeNow)
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	code := runCrmStoreCLI(s.Argv)
	os.Stdout, os.Stderr = oldOut, oldErr
	wOut.Close()
	wErr.Close()
	outB, _ := io.ReadAll(rOut)
	errB, _ := io.ReadAll(rErr)
	return xresult{code, string(outB), string(errB)}
}

// --- canonicalization ---------------------------------------------------------

var ulidRe = regexp.MustCompile(`(c_|d_|act_|draft_|rsv_)?[0-9A-HJKMNP-TV-Z]{26}`)

type canonicalizer struct {
	root   string
	ids    map[string]string
	shorts map[string]string // 10-char truncations (kanban/today-view print id[:10])
	n      int
}

func newCanon(root string) *canonicalizer {
	return &canonicalizer{root: root, ids: map[string]string{}, shorts: map[string]string{}}
}

func (cz *canonicalizer) apply(s string) string {
	s = strings.ReplaceAll(s, cz.root, "<ROOT>")
	s = ulidRe.ReplaceAllStringFunc(s, func(m string) string {
		// explicit fixture ids (c_lead1 etc.) never match the 26-char pattern
		if v, ok := cz.ids[m]; ok {
			return v
		}
		cz.n++
		prefix := ""
		if i := strings.Index(m, "_"); i >= 0 && !strings.ContainsAny(m[:i+1], "0123456789") {
			prefix = m[:i+1]
		}
		v := fmt.Sprintf("<%sID%d>", prefix, cz.n)
		cz.ids[m] = v
		// the ULID's leading chars are wall-clock ms, so a printed id[:10] is
		// only COINCIDENTALLY equal across the two runs — canonicalize it too
		if len(m) > 10 {
			cz.shorts[m[:10]] = fmt.Sprintf("<%sSID%d>", prefix, cz.n)
		}
		return v
	})
	for short, v := range cz.shorts {
		s = strings.ReplaceAll(s, short, v)
	}
	return s
}

// canonJSON parses s as JSON and re-renders deterministically (sorted keys,
// integral floats as ints) after canonicalizing embedded strings.
func (cz *canonicalizer) canonJSON(t *testing.T, s string, ignoreKeys []string) string {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		// not JSON (markdown log etc.) — canonicalize as text
		return cz.apply(s)
	}
	ign := map[string]bool{}
	for _, k := range ignoreKeys {
		ign[k] = true
	}
	var walk func(any) any
	walk = func(x any) any {
		switch y := x.(type) {
		case map[string]any:
			out := map[string]any{}
			for k, val := range y {
				if ign[k] {
					continue
				}
				out[cz.apply(k)] = walk(val)
			}
			return out
		case []any:
			for i := range y {
				y[i] = walk(y[i])
			}
			return y
		case string:
			return cz.apply(y)
		case float64:
			if y == float64(int64(y)) {
				return int64(y)
			}
			return y
		}
		return x
	}
	b, err := json.Marshal(walk(v))
	if err != nil {
		t.Fatalf("canon marshal: %v", err)
	}
	return string(b)
}

// snapshotTree returns canonical-path -> canonical-content for every compared file.
func (cz *canonicalizer) snapshotTree(t *testing.T, root string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		base := filepath.Base(rel)
		// implementation details + renderer artifacts excluded from parity
		if strings.Contains(rel, "/.locks/") || strings.HasSuffix(rel, ".html") ||
			strings.HasSuffix(rel, ".render_status.json") || // renderer sidecar (Go renderer lands in the next slice)
			strings.HasSuffix(base, ".tmp") || strings.Contains(base, ".tmp.") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		key := cz.apply(rel)
		body := string(data)
		if strings.HasSuffix(rel, ".json") {
			out[key] = cz.canonJSON(t, body, nil)
			return nil
		}
		if strings.HasSuffix(rel, ".jsonl") {
			var lines []string
			for _, line := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				lines = append(lines, cz.canonJSON(t, line, nil))
			}
			out[key] = strings.Join(lines, "\n")
			return nil
		}
		out[key] = cz.apply(body)
		return nil
	})
	return out
}

// --- fixtures -----------------------------------------------------------------

func writeFixture(t *testing.T, ws string) {
	t.Helper()
	mustWrite := func(rel, body string) {
		p := filepath.Join(ws, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("sendboxes/sendboxes.json", `{"sendboxes": [
	  {"slug": "sb-a", "email": "a@gmail.com", "domain": "gmail.com", "quota_today": 40, "status": "healthy", "imap_uid_cursor": 0},
	  {"slug": "sb-b", "email": "b@gmail.com", "domain": "gmail.com", "quota_today": 40, "status": "needs_reauth", "imap_uid_cursor": 0}
	]}`)
	// a prior send at step 1 makes lead2 bump-due later
	mustWrite("campaigns/demo/sent/2026-07/sent_log.jsonl",
		`{"lead_id": "c_lead2", "campaign": "demo", "step": 1, "sent_at": "2026-07-10T09:00:00Z", "sendbox": "sb-a", "rfc_message_id": "<m1@x>"}`+"\n")
}

// --- the scenario ---------------------------------------------------------------

func TestCrmStoreGoldenCrossValidation(t *testing.T) {
	pyTool := findCrmStorePy(t)

	pyRoot := filepath.Join(t.TempDir(), "dcp")
	goRoot := filepath.Join(t.TempDir(), "dcp")
	pyWS := filepath.Join(pyRoot, "clients", "leadup", "video_us", "outreach")
	goWS := filepath.Join(goRoot, "clients", "leadup", "video_us", "outreach")

	dossier := `{"identity": {"still_active": "confirmed", "current_company": "KW", "channels_found": {"emails": ["extra@kw.com"]}},
	 "hooks": [
	   {"type": "new_listing", "summary": "listed 123 Main St", "evidence_url": "https://z/1", "observed_date": "2026-07-14", "confidence": 0.9, "analysis": {"sensitivity": "public_business"}},
	   {"type": "award", "summary": "personal award", "evidence_url": "https://z/2", "confidence": 0.8, "analysis": {"sensitivity": "personal"}},
	   {"type": "bad", "summary": "no url", "evidence_url": "N/A", "confidence": 0.5}
	 ],
	 "writing_brief": {"one_liner": "top KW agent", "personalization_confidence": 0.85, "do_not_mention": ["kids"]}}`

	steps := []xstep{
		{FakeNow: "2026-07-19T10:00:00Z", Argv: []string{"--pipeline", "{ROOT}", "--client", "leadup", "--business", "video", "--location", "us", "init-client"}},
		{FakeNow: "2026-07-19T10:01:00Z", Argv: []string{"--client-dir", "{WS}", "contact", "add", "--json", `{"id": "c_lead1", "name": {"full": "Susan Vo"}, "identities": {"emails": [{"address": "Susan@KW.com", "is_primary": true}], "phones": [{"number": "(415) 555-0101"}]}}`}},
		{FakeNow: "2026-07-19T10:02:00Z", Argv: []string{"--client-dir", "{WS}", "contact", "add", "--json", `{"id": "c_lead2", "name": {"full": "Binh Tran"}, "identities": {"emails": [{"address": "binh@remax.com", "is_primary": true}]}}`}},
		{FakeNow: "2026-07-19T10:03:00Z", Argv: []string{"--client-dir", "{WS}", "contact", "add", "--json", `{"name": {"full": "Dup Susan"}, "identities": {"emails": [{"address": "susan@kw.com"}]}}`}}, // matched
		{FakeNow: "2026-07-19T10:04:00Z", Argv: []string{"--client-dir", "{WS}", "contact", "list", "--where", "lifecycle_stage,=,lead"}},
		{FakeNow: "2026-07-19T10:05:00Z", Argv: []string{"--client-dir", "{WS}", "segment", "set", "--json", `{"id": "all", "name": "all", "where": [["lifecycle_stage", "=", "lead"]]}`}},
		{FakeNow: "2026-07-19T10:06:00Z", Argv: []string{"--client-dir", "{WS}", "campaign", "create", "--slug", "demo", "--json", `{"audience": {"segment": "all"}, "sendboxes": ["sb-a", "sb-b"], "daily_quota": 10}`}},
		{FakeNow: "2026-07-19T10:07:00Z", Argv: []string{"--client-dir", "{WS}", "campaign", "queue", "--slug", "demo"}},
		{FakeNow: "2026-07-19T10:08:00Z", Argv: []string{"--client-dir", "{WS}", "enrich", "status", "--contact", "c_lead1"}},
		{FakeNow: "2026-07-19T10:09:00Z", Argv: []string{"--client-dir", "{WS}", "enrich", "due", "--campaign", "demo"}},
		{FakeNow: "2026-07-19T10:10:00Z", Argv: []string{"--client-dir", "{WS}", "enrich", "write", "--contact", "c_lead1", "--campaign", "demo", "--json", dossier}},
		{FakeNow: "2026-07-19T10:11:00Z", Argv: []string{"--client-dir", "{WS}", "enrich", "get", "--contact", "c_lead1"}},
		{FakeNow: "2026-07-19T10:12:00Z", Argv: []string{"--client-dir", "{WS}", "draft", "write", "--contact", "c_lead1", "--campaign", "demo", "--json", `{"step": 1, "subject": "Idea for 123 Main St", "body_text": "Hi Susan...", "hooks_used": [{"type": "new_listing", "evidence_url": "https://z/1"}]}`}},
		{FakeNow: "2026-07-19T10:13:00Z", Argv: []string{"--client-dir", "{WS}", "draft", "write", "--contact", "c_lead1", "--campaign", "demo", "--json", `{"step": 1, "subject": "Re: hello", "body_text": "x", "hooks_used": [{"type": "new_listing", "evidence_url": "https://z/1"}]}`},
			WantFail: "step-1 subject must not begin with Re:/Fwd:"},
		{FakeNow: "2026-07-19T10:14:00Z", Argv: []string{"--client-dir", "{WS}", "draft", "budget", "--campaign", "demo"}},
		{FakeNow: "2026-07-19T10:15:00Z", Argv: []string{"--client-dir", "{WS}", "draft", "list", "--campaign", "demo"}},
		{FakeNow: "2026-07-19T10:16:00Z", Argv: []string{"--client-dir", "{WS}", "approval-report"}},
		{FakeNow: "2026-07-19T10:17:00Z", Argv: []string{"--client-dir", "{WS}", "approve", "--json", `{"edit": [{"n": 1, "subject": "Better idea for 123 Main St"}], "approve": "1"}`}},
		{FakeNow: "2026-07-19T10:18:00Z", Argv: []string{"--client-dir", "{WS}", "followups", "due", "--campaign", "demo"}},
		{FakeNow: "2026-07-19T10:19:00Z", Argv: []string{"--client-dir", "{WS}", "apply-rules", "--event", "reply_positive", "--contact", "c_lead1", "--activity", "act_fixture01"}},
		{FakeNow: "2026-07-19T10:20:00Z", Argv: []string{"--client-dir", "{WS}", "apply-rules", "--event", "reply_positive", "--contact", "c_lead1", "--activity", "act_fixture01"}}, // idempotent
		{FakeNow: "2026-07-19T10:21:00Z", Argv: []string{"--client-dir", "{WS}", "suppress", "add", "--kind", "email", "--value", "Spam@X.com", "--reason", "unsubscribe", "--tag", "test_fixture"}},
		{FakeNow: "2026-07-19T10:22:00Z", Argv: []string{"--client-dir", "{WS}", "suppress", "check", "--email", "spam@x.com"}},
		{FakeNow: "2026-07-19T10:23:00Z", Argv: []string{"--client-dir", "{WS}", "task", "add", "--json", `{"title": "Call Susan", "contact_id": "c_lead1", "due_at": "2026-07-20T10:00:00Z"}`}},
		{FakeNow: "2026-07-19T10:24:00Z", Argv: []string{"--client-dir", "{WS}", "reserve", "--sendbox", "sb-a", "--day", "2026-07-19", "--cap", "2"}},
		{FakeNow: "2026-07-19T10:25:00Z", Argv: []string{"--client-dir", "{WS}", "reserve", "--sendbox", "sb-a", "--day", "2026-07-19", "--cap", "2"}},
		{FakeNow: "2026-07-19T10:26:00Z", Argv: []string{"--client-dir", "{WS}", "reserve", "--sendbox", "sb-a", "--day", "2026-07-19", "--cap", "2"}}, // denied
		{FakeNow: "2026-07-19T10:27:00Z", Argv: []string{"--client-dir", "{WS}", "contact", "merge", "--loser", "c_lead2", "--winner", "c_lead1"}},
		{FakeNow: "2026-07-19T10:28:00Z", Argv: []string{"--client-dir", "{WS}", "validate", "--rebuild-index"}},
		{FakeNow: "2026-07-19T10:29:00Z", Argv: []string{"--client-dir", "{WS}", "today-view"}},
		{FakeNow: "2026-07-19T10:30:00Z", Argv: []string{"--client-dir", "{WS}", "kanban"}},
		{FakeNow: "2026-07-19T10:31:00Z", Argv: []string{"--client-dir", "{WS}", "weekly-report"}},
		{FakeNow: "2026-07-19T10:32:00Z", Argv: []string{"--client-dir", "{WS}", "monthly-report", "--month", "2026-07"}},
	}

	subst := func(argv []string, root, ws string) []string {
		out := make([]string, len(argv))
		for i, a := range argv {
			a = strings.ReplaceAll(a, "{ROOT}", root)
			a = strings.ReplaceAll(a, "{WS}", ws)
			out[i] = a
		}
		return out
	}

	pyCanon := newCanon(pyRoot)
	goCanon := newCanon(goRoot)

	for i, s := range steps {
		if i == 1 { // after init-client created the workspaces
			writeFixture(t, pyWS)
			writeFixture(t, goWS)
			// ui_inbox decision applied later needs a real draft id — the ingest-ui path
			// is already covered by the Python and Go unit suites; the xval scenario
			// covers everything id-independent.
		}
		pyStep := s
		pyStep.Argv = subst(s.Argv, pyRoot, pyWS)
		goStep := s
		goStep.Argv = subst(s.Argv, goRoot, goWS)
		pr := runPyStep(t, pyTool, pyStep)
		gr := runGoStep(t, goStep)

		if s.WantFail != "" {
			if pr.Code == 0 || gr.Code == 0 {
				t.Fatalf("step %d %v: expected both to fail (py=%d go=%d)", i, s.Argv, pr.Code, gr.Code)
			}
			if !strings.Contains(pr.Stderr, s.WantFail) || !strings.Contains(gr.Stderr, s.WantFail) {
				t.Fatalf("step %d: stderr mismatch\npy: %s\ngo: %s", i, pr.Stderr, gr.Stderr)
			}
			continue
		}
		if pr.Code != gr.Code {
			t.Fatalf("step %d %v: exit py=%d go=%d\npy-err: %s\ngo-err: %s", i, s.Argv, pr.Code, gr.Code, pr.Stderr, gr.Stderr)
		}
		pc := pyCanon.canonJSON(t, pr.Stdout, s.IgnoreKey)
		gc := goCanon.canonJSON(t, gr.Stdout, s.IgnoreKey)
		if pc != gc {
			t.Fatalf("step %d %v: stdout mismatch\npy: %s\ngo: %s", i, s.Argv, pc, gc)
		}
	}

	// final tree parity
	pyTree := pyCanon.snapshotTree(t, pyRoot)
	goTree := goCanon.snapshotTree(t, goRoot)
	var pyKeys, goKeys []string
	for k := range pyTree {
		pyKeys = append(pyKeys, k)
	}
	for k := range goTree {
		goKeys = append(goKeys, k)
	}
	sort.Strings(pyKeys)
	sort.Strings(goKeys)
	if strings.Join(pyKeys, "\n") != strings.Join(goKeys, "\n") {
		t.Fatalf("tree file sets differ\npy-only: %v\ngo-only: %v",
			diffKeys(pyKeys, goKeys), diffKeys(goKeys, pyKeys))
	}
	for _, k := range pyKeys {
		if pyTree[k] != goTree[k] {
			t.Errorf("tree content mismatch at %s\npy: %s\ngo: %s", k, clip(pyTree[k]), clip(goTree[k]))
		}
	}
}

func diffKeys(a, b []string) []string {
	set := map[string]bool{}
	for _, k := range b {
		set[k] = true
	}
	var out []string
	for _, k := range a {
		if !set[k] {
			out = append(out, k)
		}
	}
	return out
}

func clip(s string) string {
	if len(s) > 600 {
		return s[:600] + "…"
	}
	return s
}

func TestImportLeadsGoldenCrossValidation(t *testing.T) {
	pyRepo := findCrmStorePy(t)
	pyTool := filepath.Join(filepath.Dir(pyRepo), "import_leads.py")
	if _, err := os.Stat(pyTool); err != nil {
		t.Skip("import_leads.py not present")
	}

	pyRoot := filepath.Join(t.TempDir(), "dcp")
	goRoot := filepath.Join(t.TempDir(), "dcp")
	pyWS := filepath.Join(pyRoot, "clients", "leadup", "video_us", "outreach")
	goWS := filepath.Join(goRoot, "clients", "leadup", "video_us", "outreach")

	csvBody := "Full Name,Email,Cell Phone,Office Name,Website\n" +
		"Susan Vo,susan@kw.com,(415) 555-0101,KW Bay Area,https://susanvo.com\n" +
		"Binh Tran,BINH@remax.com,,RE/MAX,\n" +
		"No Identity,,,,\n" +
		"Susan Dup,susan@kw.com,,KW,\n" +
		"Spam Guy,spam@x.com,,X,\n"
	csvPath := filepath.Join(t.TempDir(), "leads.csv")
	if err := os.WriteFile(csvPath, []byte(csvBody), 0o644); err != nil {
		t.Fatal(err)
	}

	steps := []xstep{
		{FakeNow: "2026-07-19T11:00:00Z", Argv: []string{"crm", "--pipeline", "{ROOT}", "--client", "leadup", "--business", "video", "--location", "us", "init-client"}},
		{FakeNow: "2026-07-19T11:01:00Z", Argv: []string{"crm", "--client-dir", "{WS}", "suppress", "add", "--kind", "email", "--value", "spam@x.com", "--reason", "unsubscribe"}},
		{FakeNow: "2026-07-19T11:02:00Z", Argv: []string{"leads", "inspect", "--file", csvPath}},
		{FakeNow: "2026-07-19T11:03:00Z", Argv: []string{"leads", "import", "--client-dir", "{WS}", "--file", csvPath, "--list-slug", "realtors", "--no-mx-check"}},
		{FakeNow: "2026-07-19T11:04:00Z", Argv: []string{"leads", "import", "--client-dir", "{WS}", "--file", csvPath, "--list-slug", "realtors", "--no-mx-check"}}, // idempotent no-op
		{FakeNow: "2026-07-19T11:05:00Z", Argv: []string{"crm", "--client-dir", "{WS}", "contact", "list", "--where", "lifecycle_stage,=,lead"}},
	}

	pyCanon := newCanon(pyRoot)
	goCanon := newCanon(goRoot)
	// the CSV lives outside both roots; canonicalize its temp path too
	csvDir := filepath.Dir(csvPath)

	for i, s := range steps {
		pyArgv := make([]string, len(s.Argv))
		goArgv := make([]string, len(s.Argv))
		for j, a := range s.Argv {
			pyArgv[j] = strings.ReplaceAll(strings.ReplaceAll(a, "{ROOT}", pyRoot), "{WS}", pyWS)
			goArgv[j] = strings.ReplaceAll(strings.ReplaceAll(a, "{ROOT}", goRoot), "{WS}", goWS)
		}
		var pr, gr xresult
		if pyArgv[0] == "crm" {
			pr = runPyStep(t, pyRepo, xstep{FakeNow: s.FakeNow, Argv: pyArgv[1:]})
			gr = runGoStep(t, xstep{FakeNow: s.FakeNow, Argv: append([]string{}, goArgv[1:]...)})
		} else {
			pr = runPyStep(t, pyTool, xstep{FakeNow: s.FakeNow, Argv: pyArgv[1:]})
			gr = func() xresult {
				t.Setenv("OUTREACHCRM_TEST_MODE", "1")
				t.Setenv("OUTREACHCRM_FAKE_NOW", s.FakeNow)
				oldOut, oldErr := os.Stdout, os.Stderr
				rOut, wOut, _ := os.Pipe()
				rErr, wErr, _ := os.Pipe()
				os.Stdout, os.Stderr = wOut, wErr
				code := runImportLeadsCLI(goArgv[1:])
				os.Stdout, os.Stderr = oldOut, oldErr
				wOut.Close()
				wErr.Close()
				outB, _ := io.ReadAll(rOut)
				errB, _ := io.ReadAll(rErr)
				return xresult{code, string(outB), string(errB)}
			}()
		}
		if pr.Code != gr.Code {
			t.Fatalf("leads step %d %v: exit py=%d go=%d\npy-err: %s\ngo-err: %s", i, s.Argv, pr.Code, gr.Code, pr.Stderr, gr.Stderr)
		}
		pc := strings.ReplaceAll(pyCanon.canonJSON(t, pr.Stdout, nil), csvDir, "<TMP>")
		gc := strings.ReplaceAll(goCanon.canonJSON(t, gr.Stdout, nil), csvDir, "<TMP>")
		if pc != gc {
			t.Fatalf("leads step %d %v: stdout mismatch\npy: %s\ngo: %s", i, s.Argv, pc, gc)
		}
	}

	pyTree := pyCanon.snapshotTree(t, pyRoot)
	goTree := goCanon.snapshotTree(t, goRoot)
	var keys []string
	for k := range pyTree {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if _, ok := goTree[k]; !ok {
			t.Errorf("go tree missing %s", k)
			continue
		}
		pv := strings.ReplaceAll(pyTree[k], csvDir, "<TMP>")
		gv := strings.ReplaceAll(goTree[k], csvDir, "<TMP>")
		if pv != gv {
			t.Errorf("leads tree mismatch at %s\npy: %s\ngo: %s", k, clip(pv), clip(gv))
		}
	}
	if len(goTree) != len(pyTree) {
		t.Errorf("tree sizes differ: py=%d go=%d", len(pyTree), len(goTree))
	}
}
