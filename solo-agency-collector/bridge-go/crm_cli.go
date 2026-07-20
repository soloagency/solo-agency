package main

// crm_cli.go — `<bridge> tool crm-store ...`, argparse-compatible with
// outreach/tools/crm_store.py: same flags (any position), same subcommands,
// same JSON-to-stdout shape, same exit codes (0 ok, 1 StorageError, 2 usage).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type cliArgs struct {
	flags map[string][]string // --flag -> values (repeatable flags accumulate)
	bools map[string]bool
	pos   []string
}

// parseCLIArgs consumes known value-flags and bool-flags from anywhere in argv,
// like argparse. Unknown --flags are an error (exit 2).
func parseCLIArgs(args []string, valueFlags, boolFlags map[string]bool) (*cliArgs, error) {
	out := &cliArgs{flags: map[string][]string{}, bools: map[string]bool{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			name := a
			var val string
			hasInline := false
			if j := strings.Index(a, "="); j >= 0 {
				name, val, hasInline = a[:j], a[j+1:], true
			}
			if boolFlags[name] && !hasInline {
				out.bools[name] = true
				continue
			}
			if !valueFlags[name] {
				return nil, fmt.Errorf("unrecognized argument: %s", name)
			}
			if !hasInline {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("argument %s: expected one argument", name)
				}
				i++
				val = args[i]
			}
			out.flags[name] = append(out.flags[name], val)
			continue
		}
		out.pos = append(out.pos, a)
	}
	return out, nil
}

func (a *cliArgs) get(name string) string {
	if vs := a.flags[name]; len(vs) > 0 {
		return vs[len(vs)-1]
	}
	return ""
}

func (a *cliArgs) getInt(name string, def int) int {
	if s := a.get(name); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return def
}

func crmOut(obj any, code int) int {
	fmt.Println(marshalIndentJSON(obj))
	return code
}

func crmUsageErr(msg string) int {
	fmt.Fprintln(os.Stderr, "crm-store:", msg)
	return 2
}

func crmFail(err error) int {
	if _, ok := err.(*storageError); ok {
		fmt.Fprintln(os.Stderr, "StorageError:", err.Error())
		return 1
	}
	fmt.Fprintln(os.Stderr, "crm-store:", err.Error())
	return 1
}

func resolveClientDirCLI(pipeline, client, clientDir, business, location string, create bool) (string, error) {
	if clientDir != "" {
		abs, err := filepath.Abs(clientDir)
		if err != nil {
			return clientDir, nil
		}
		return abs, nil
	}
	if pipeline == "" || client == "" {
		return "", fmt.Errorf("need --client-dir, or --pipeline and --client")
	}
	absP, err := filepath.Abs(pipeline)
	if err != nil {
		absP = pipeline
	}
	base := filepath.Join(absP, "clients", client)
	if create {
		b, l := business, location
		if b == "" {
			b = "main"
		}
		if l == "" {
			l = "main"
		}
		ws := filepath.Join(base, b+"_"+l, "outreach")
		if err := os.MkdirAll(ws, 0o755); err != nil {
			return "", err
		}
		cfg := filepath.Join(absP, "storage_config.json")
		if _, err := os.Stat(cfg); err != nil {
			if err := os.MkdirAll(absP, 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(cfg, []byte(`{"backend": "json"}`), 0o644); err != nil {
				return "", err
			}
		}
		return ws, nil
	}
	var matches []string
	entries, _ := os.ReadDir(base)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		d := filepath.Join(base, name, "outreach")
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			matches = append(matches, d)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no outreach workspace under %s; run init-client first", base)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple workspaces under %s; pass --client-dir explicitly", base)
	}
	return matches[0], nil
}

func parseJSONArg(s string) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, fmt.Errorf("bad --json: %v", err)
	}
	return m, nil
}

// runCrmStoreCLI dispatches `tool crm-store <args>`; returns the process exit code.
func runCrmStoreCLI(args []string) int {
	valueFlags := map[string]bool{}
	for _, f := range []string{"--pipeline", "--client", "--client-dir", "--business", "--location",
		"--json", "--id", "--loser", "--winner", "--where", "--file", "--kind", "--value", "--reason",
		"--tier", "--tag", "--email", "--phone", "--contact", "--stage", "--evidence", "--sendbox",
		"--day", "--cap", "--event", "--activity", "--events", "--slug", "--limit", "--campaign",
		"--client-name", "--days", "--month", "--other"} {
		valueFlags[f] = true
	}
	boolFlags := map[string]bool{"--confirm": true, "--rebuild-index": true}
	a, err := parseCLIArgs(args, valueFlags, boolFlags)
	if err != nil {
		return crmUsageErr(err.Error())
	}
	if len(a.pos) == 0 {
		return crmUsageErr("a subcommand is required")
	}
	cmd := a.pos[0]
	op := ""
	if len(a.pos) > 1 {
		op = a.pos[1]
	}
	pipeline, client, clientDir := a.get("--pipeline"), a.get("--client"), a.get("--client-dir")
	business, location := a.get("--business"), a.get("--location")

	if cmd == "init-client" {
		cdir, err := resolveClientDirCLI(pipeline, client, clientDir, business, location, true)
		if err != nil {
			return crmUsageErr(err.Error())
		}
		store := newCrmStore(cdir)
		if err := store.initTree(); err != nil {
			return crmFail(err)
		}
		return crmOut(map[string]any{"ok": true, "client_dir": cdir, "pipelines": "default_sales"}, 0)
	}

	cdir, err := resolveClientDirCLI(pipeline, client, clientDir, "", "", false)
	if err != nil {
		return crmUsageErr(err.Error())
	}

	if cmd == "reset-client" {
		if !a.bools["--confirm"] {
			return crmOut(map[string]any{"error": "reset-client requires --confirm"}, 2)
		}
		res, err := resetClient(cdir)
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	}

	store := newCrmStore(cdir)

	switch cmd {
	case "validate":
		res, err := store.validate(a.bools["--rebuild-index"])
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	case "campaign":
		switch op {
		case "create":
			cfg := map[string]any{}
			if s := a.get("--json"); s != "" {
				if cfg, err = parseJSONArg(s); err != nil {
					return crmUsageErr(err.Error())
				}
			}
			res, err := store.createCampaign(a.get("--slug"), cfg)
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		case "get":
			if cfg := store.getCampaign(a.get("--slug")); cfg != nil {
				return crmOut(cfg, 0)
			}
			return crmOut(map[string]any{"error": "not found"}, 0)
		case "list":
			return crmOut(listAny(store.listCampaigns()), 0)
		case "queue":
			res, err := store.queueCampaign(a.get("--slug"), a.getInt("--limit", 100))
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		case "update":
			patch, err := parseJSONArg(a.get("--json"))
			if err != nil {
				return crmUsageErr(err.Error())
			}
			res, err := store.campaignUpdate(a.get("--slug"), patch)
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		}
	case "segment":
		switch op {
		case "set":
			seg, err := parseJSONArg(a.get("--json"))
			if err != nil {
				return crmUsageErr(err.Error())
			}
			res, err := store.setSegment(seg)
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		case "get", "list":
			return crmOut(store.getSegments(), 0)
		case "resolve":
			rows, err := store.resolveSegment(a.get("--id"))
			if err != nil {
				return crmFail(err)
			}
			out := []any{}
			for _, ct := range rows {
				out = append(out, map[string]any{"id": ct["id"], "name": mStr(mMap(ct, "name"), "full")})
			}
			return crmOut(out, 0)
		}
	case "enrich":
		switch op {
		case "status":
			return crmOut(store.enrichStatus(a.get("--contact"), ""), 0)
		case "due":
			return crmOut(listAny(store.enrichDue(a.get("--campaign"), a.getInt("--limit", 100), "")), 0)
		case "write":
			dossier, err := parseJSONArg(a.get("--json"))
			if err != nil {
				return crmUsageErr(err.Error())
			}
			res, err := store.enrichWrite(a.get("--contact"), dossier, a.get("--campaign"))
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		case "get":
			ct := store.getContact(a.get("--contact"))
			en := mMap(ct, "enrichment")
			if en == nil {
				en = map[string]any{}
			}
			return crmOut(en, 0)
		}
	case "draft":
		switch op {
		case "write":
			d, err := parseJSONArg(a.get("--json"))
			if err != nil {
				return crmUsageErr(err.Error())
			}
			res, err := store.draftWrite(a.get("--contact"), a.get("--campaign"), draftArgs{
				Step: mInt(d, "step", 1), Subject: mStr(d, "subject"), BodyText: mStr(d, "body_text"),
				BodyHTML: mStr(d, "body_html"), HooksUsed: mapsOf(mList(d, "hooks_used")),
				Tracking: mStr(d, "tracking"), IsReply: truthy(d["is_reply"]),
				BankMessagesUsed: mList(d, "bank_messages_used"), CompanionURL: mStr(d, "companion_url")})
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		case "list":
			return crmOut(listAny(store.listPendingDrafts(a.get("--campaign"))), 0)
		case "budget":
			res, err := store.draftBudget(a.get("--campaign"), a.get("--day"))
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		}
	case "approval-report":
		res, err := store.renderApprovalReport(a.get("--campaign"))
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	case "approve":
		actions, err := parseJSONArg(a.get("--json"))
		if err != nil {
			return crmUsageErr(err.Error())
		}
		res, err := store.approveApply(actions, "human")
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	case "ingest-ui":
		res, err := store.ingestUIDecisions()
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	case "followups":
		if op == "due" {
			res, err := store.followupsDue(a.get("--campaign"), "")
			if err != nil {
				return crmFail(err)
			}
			return crmOut(listAny(res), 0)
		}
	case "today-view":
		res, err := store.renderTodayView("")
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	case "kanban":
		res, err := store.renderKanban("")
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	case "weekly-report":
		res, err := store.renderWeeklyReport("", a.get("--client-name"), a.getInt("--days", 7))
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	case "monthly-report":
		res, err := store.renderMonthlyReport("", a.get("--client-name"), a.get("--month"))
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	case "contact":
		switch op {
		case "add":
			fields, err := parseJSONArg(a.get("--json"))
			if err != nil {
				return crmUsageErr(err.Error())
			}
			leadID, outcome, err := store.addContact(fields)
			if err != nil {
				return crmFail(err)
			}
			return crmOut(map[string]any{"lead_id": leadID, "outcome": outcome}, 0)
		case "get":
			if ct := store.getContact(a.get("--id")); ct != nil {
				return crmOut(ct, 0)
			}
			return crmOut(map[string]any{"error": "not found"}, 0)
		case "list":
			var where []cond
			for _, w := range a.flags["--where"] {
				parts := strings.SplitN(w, ",", 3)
				if len(parts) != 3 {
					return crmUsageErr(fmt.Sprintf("--where must be field,op,value (got %q)", w))
				}
				where = append(where, cond{Field: parts[0], Op: parts[1], Value: parts[2]})
			}
			rows, err := store.a.query("contacts", where, nil, -1, 0)
			if err != nil {
				return crmFail(err)
			}
			return crmOut(listAny(rows), 0)
		case "merge":
			res, err := store.merge(a.get("--loser"), a.get("--winner"))
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		case "unsuspect":
			res, err := store.clearDuplicateSuspect(a.get("--id"), a.get("--other"))
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		}
	case "activity":
		ev, err := parseJSONArg(a.get("--json"))
		if err != nil {
			return crmUsageErr(err.Error())
		}
		res, err := store.logActivity(mStr(ev, "type"), mStr(ev, "contact_id"), mStr(ev, "summary"),
			strOr(mStr(ev, "by"), "agent"), ev["deal_id"], mMap(ev, "ref"))
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	case "pipeline":
		switch op {
		case "get":
			return crmOut(store.getPipelines(), 0)
		case "ensure-default":
			res, err := store.ensureDefaultPipelines()
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		case "set":
			m, err := readJSONFile(a.get("--file"))
			if err != nil {
				return crmFail(err)
			}
			if err := store.setPipelines(m); err != nil {
				return crmFail(err)
			}
			return crmOut(map[string]any{"ok": true}, 0)
		}
	case "suppress":
		switch op {
		case "add":
			tags := []any{}
			for _, t := range a.flags["--tag"] {
				tags = append(tags, t)
			}
			tier := strOr(a.get("--tier"), "client")
			res, err := store.suppressAdd(a.get("--kind"), a.get("--value"), a.get("--reason"),
				tier, nil, "", "human", tags)
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		case "check":
			hit := store.isSuppressed(a.get("--email"), a.get("--phone"), nil)
			out := map[string]any{"suppressed": hit != nil, "match": nil}
			if hit != nil {
				out["match"] = hit
			}
			return crmOut(out, 0)
		}
	case "deal":
		switch op {
		case "create":
			extra := map[string]any{}
			if s := a.get("--json"); s != "" {
				if extra, err = parseJSONArg(s); err != nil {
					return crmUsageErr(err.Error())
				}
			}
			stage := strOr(a.get("--stage"), "new_reply")
			res, err := store.createDeal(a.get("--contact"), stage, "default_sales", "human", "", extra)
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		case "move":
			res, err := store.moveDeal(a.get("--id"), a.get("--stage"), a.get("--evidence"), "human")
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		}
	case "task":
		switch op {
		case "add":
			t, err := parseJSONArg(a.get("--json"))
			if err != nil {
				return crmUsageErr(err.Error())
			}
			res, err := store.addTask(mStr(t, "title"), t["contact_id"], t["deal_id"],
				mStr(t, "due_at"), strOr(mStr(t, "created_by"), "human"), "")
			if err != nil {
				return crmFail(err)
			}
			return crmOut(res, 0)
		case "done":
			latest := map[string]map[string]any{}
			rows, _ := store.a.readLog("tasks", -1, nil)
			for _, x := range rows {
				latest[mStr(x, "id")] = x
			}
			if t, ok := latest[a.get("--id")]; ok {
				cp := map[string]any{}
				for k, v := range t {
					cp[k] = v
				}
				cp["status"] = "done"
				res, err := store.a.appendLog("tasks", cp)
				if err != nil {
					return crmFail(err)
				}
				return crmOut(res, 0)
			}
			return crmOut(map[string]any{"error": "task not found"}, 2)
		}
	case "reserve":
		capN := a.getInt("--cap", -1)
		if a.get("--sendbox") == "" || a.get("--day") == "" || capN < 0 {
			return crmUsageErr("reserve needs --sendbox, --day, --cap")
		}
		tok, err := store.a.reserve(a.get("--sendbox"), a.get("--day"), capN)
		if err != nil {
			return crmFail(err)
		}
		var tokOut any
		if tok != "" {
			tokOut = tok
		}
		return crmOut(map[string]any{"token": tokOut, "granted": tok != "",
			"count": store.a.reservationCount(a.get("--sendbox"), a.get("--day"))}, 0)
	case "apply-rules":
		var events []map[string]any
		if f := a.get("--events"); f != "" {
			data, err := os.ReadFile(f)
			if err != nil {
				return crmFail(err)
			}
			var raw []any
			if err := json.Unmarshal(data, &raw); err != nil {
				return crmFail(err)
			}
			events = mapsOf(raw)
		} else if a.get("--event") != "" && a.get("--contact") != "" {
			events = []map[string]any{{"type": a.get("--event"), "contact_id": a.get("--contact"),
				"activity_id": a.get("--activity")}}
		} else {
			return crmOut(map[string]any{"error": "need --events or (--event and --contact)"}, 2)
		}
		res, err := store.applyRules(events)
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	}
	return crmOut(map[string]any{"error": "unknown command"}, 2)
}

func resetClient(clientDir string) (map[string]any, error) {
	removed := []any{}
	for _, sub := range []string{"crm/contacts", "crm/accounts", "crm/deals", "crm/activities",
		"crm/tasks", "crm/.seq", "crm/.locks", "crm/contact_identities.jsonl",
		"crm/suppression.jsonl", "sendboxes/_reservations", "lists", "campaigns", "inbox_sync", "outputs"} {
		p := filepath.Join(clientDir, filepath.FromSlash(sub))
		if st, err := os.Stat(p); err == nil {
			if st.IsDir() {
				if err := os.RemoveAll(p); err != nil {
					return nil, err
				}
			} else if err := os.Remove(p); err != nil {
				return nil, err
			}
			removed = append(removed, sub)
		}
	}
	if err := newCrmStore(clientDir).initTree(); err != nil {
		return nil, err
	}
	return map[string]any{"reset": true, "client_dir": clientDir, "removed": removed}, nil
}

func listAny[T any](in []T) []any {
	out := []any{}
	for _, v := range in {
		out = append(out, v)
	}
	return out
}

func strOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
