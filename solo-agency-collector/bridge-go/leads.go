package main

// leads.go — Go port of outreach/tools/import_leads.py: CSV/TXT/TSV/XLSX lead
// lists -> contacts (deduped via the identity index, suppression-checked),
// with the same lists/{slug}/{leads.jsonl,list_manifest.json,import_log.md}
// outputs. Idempotency keys are byte-identical to Python's
// sha256(file || json.dumps(mapping, sort_keys=True)) so a list imported by
// one implementation is a no-op in the other.

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var leadSynonymOrder = []string{"email", "full_name", "first_name", "last_name", "company",
	"phone", "website", "city", "state", "facebook", "linkedin", "instagram"}

var leadSynonyms = map[string][]string{
	"email":      {"email", "e-mail", "email address", "mail"},
	"full_name":  {"full name", "name", "contact name", "fullname"},
	"first_name": {"first name", "first", "firstname"},
	"last_name":  {"last name", "last", "lastname", "surname"},
	"company":    {"company", "office name", "brokerage", "organization", "org", "business"},
	"phone":      {"cell phone", "mobile", "cell", "phone", "phone number", "office phone"},
	"website":    {"website", "url", "web", "site"},
	"city":       {"city", "office city"},
	"state":      {"state", "office state"},
	"facebook":   {"facebook", "fb"},
	"linkedin":   {"linkedin"},
	"instagram":  {"instagram", "ig"},
}

var colLettersRe = regexp.MustCompile(`^[A-Z]{1,2}$`)

func looksLikeLettersHeader(headers []string) bool {
	var vals []string
	for _, h := range headers {
		v := strings.ToUpper(strings.TrimSpace(h))
		if v != "" {
			vals = append(vals, v)
		}
	}
	if len(vals) < 3 {
		return false
	}
	for _, v := range vals {
		if !colLettersRe.MatchString(v) {
			return false
		}
	}
	return true
}

// sniffDelimiter picks among , \t ; by the most consistent per-line count in
// the sample (a stand-in for csv.Sniffer restricted to those delimiters).
func sniffDelimiter(sample string) rune {
	lines := []string{}
	for _, ln := range strings.Split(sample, "\n") {
		if strings.TrimSpace(ln) != "" {
			lines = append(lines, ln)
		}
		if len(lines) >= 10 {
			break
		}
	}
	best, bestScore := ',', -1
	for _, d := range []rune{',', '\t', ';'} {
		counts := map[int]int{}
		for _, ln := range lines {
			counts[strings.Count(ln, string(d))]++
		}
		score := -1
		for c, freq := range counts {
			if c > 0 && freq > score {
				score = freq * (c + 1)
			}
		}
		if score > bestScore {
			best, bestScore = d, score
		}
	}
	return best
}

// leadRows streams (headers, next()) from a csv/txt/tsv/xlsx file.
func leadRows(path string) ([]string, func() (map[string]string, bool), func(), error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv", ".txt", ".tsv":
		f, err := os.Open(path)
		if err != nil {
			return nil, nil, nil, err
		}
		sample := make([]byte, 8192)
		n, _ := f.Read(sample)
		sample = sample[:n]
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			f.Close()
			return nil, nil, nil, err
		}
		text := strings.TrimPrefix(string(sample), "\uFEFF")
		var lines []string
		for _, ln := range strings.Split(text, "\n") {
			if s := strings.TrimSpace(ln); s != "" {
				lines = append(lines, s)
			}
		}
		single := 0
		for _, ln := range lines {
			if !strings.ContainsAny(ln, ",\t; ") {
				single++
			}
		}
		threshold := int(0.8 * float64(len(lines)))
		if threshold < 1 {
			threshold = 1
		}
		if ext == ".txt" && len(lines) > 0 && single >= threshold {
			// one-email-per-line
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				return nil, nil, nil, err
			}
			all := strings.Split(strings.TrimPrefix(string(data), "\uFEFF"), "\n")
			i := 0
			next := func() (map[string]string, bool) {
				for i < len(all) {
					ln := strings.TrimSpace(all[i])
					i++
					if ln != "" {
						return map[string]string{"email": ln}, true
					}
				}
				return nil, false
			}
			return []string{"email"}, next, func() {}, nil
		}
		r := csv.NewReader(stripBOMReader(f))
		r.Comma = sniffDelimiter(text)
		r.FieldsPerRecord = -1
		r.LazyQuotes = true
		headerRec, err := r.Read()
		if err != nil {
			f.Close()
			if err == io.EOF {
				empty := func() (map[string]string, bool) { return nil, false }
				return nil, empty, func() {}, nil
			}
			return nil, nil, nil, err
		}
		headers := headerRec
		realHeaders := headers
		if looksLikeLettersHeader(headers) {
			if first, err := r.Read(); err == nil {
				realHeaders = make([]string, len(headers))
				for i := range headers {
					v := ""
					if i < len(first) {
						v = strings.TrimSpace(first[i])
					}
					realHeaders[i] = v
				}
			}
		}
		next := func() (map[string]string, bool) {
			rec, err := r.Read()
			if err != nil {
				return nil, false
			}
			row := map[string]string{}
			for i, h := range realHeaders {
				if h == "" {
					continue
				}
				v := ""
				if i < len(rec) {
					v = rec[i]
				}
				row[h] = v
			}
			return row, true
		}
		return realHeaders, next, func() { f.Close() }, nil
	case ".xlsx":
		headers, rows, err := rowsFromXLSX(path)
		if err != nil {
			return nil, nil, nil, err
		}
		i := 0
		next := func() (map[string]string, bool) {
			if i >= len(rows) {
				return nil, false
			}
			r := rows[i]
			i++
			return r, true
		}
		return headers, next, func() {}, nil
	}
	return nil, nil, nil, fmt.Errorf("unsupported file type %s; use csv/txt/xlsx", pyRepr(ext))
}

func stripBOMReader(f *os.File) io.Reader {
	br := make([]byte, 3)
	n, _ := f.Read(br)
	if n == 3 && br[0] == 0xEF && br[1] == 0xBB && br[2] == 0xBF {
		return f
	}
	f.Seek(0, io.SeekStart)
	return f
}

// --- minimal XLSX reader (zip+XML, first workbook sheet, shared+inline strings) ---

type xlsxSST struct {
	SIs []xlsxSI `xml:"si"`
}
type xlsxSI struct {
	Ts []string `xml:"t"`
	R  []struct {
		T string `xml:"t"`
	} `xml:"r"`
}
type xlsxWorkbook struct {
	Sheets struct {
		Sheet []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheet"`
	} `xml:"sheets"`
}
type xlsxRels struct {
	Rel []struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}
type xlsxSheet struct {
	Rows []struct {
		Cs []xlsxCell `xml:"c"`
	} `xml:"sheetData>row"`
}
type xlsxCell struct {
	R  string `xml:"r,attr"`
	T  string `xml:"t,attr"`
	V  string `xml:"v"`
	IS struct {
		Ts []string `xml:"t"`
		R  []struct {
			T string `xml:"t"`
		} `xml:"r"`
	} `xml:"is"`
}

func rowsFromXLSX(path string) ([]string, []map[string]string, error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return nil, nil, err
	}
	defer z.Close()
	files := map[string]*zip.File{}
	var names []string
	for _, f := range z.File {
		files[f.Name] = f
		names = append(names, f.Name)
	}
	sort.Strings(names)
	readXML := func(name string, into any) error {
		f, ok := files[name]
		if !ok {
			return fmt.Errorf("missing %s", name)
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return xml.NewDecoder(rc).Decode(into)
	}
	var shared []string
	if _, ok := files["xl/sharedStrings.xml"]; ok {
		var sst xlsxSST
		if err := readXML("xl/sharedStrings.xml", &sst); err == nil {
			for _, si := range sst.SIs {
				var sb strings.Builder
				for _, t := range si.Ts {
					sb.WriteString(t)
				}
				for _, r := range si.R {
					sb.WriteString(r.T)
				}
				shared = append(shared, sb.String())
			}
		}
	}
	sheetName := ""
	if _, ok := files["xl/workbook.xml"]; ok {
		if _, ok2 := files["xl/_rels/workbook.xml.rels"]; ok2 {
			var wb xlsxWorkbook
			var rels xlsxRels
			if readXML("xl/workbook.xml", &wb) == nil && readXML("xl/_rels/workbook.xml.rels", &rels) == nil &&
				len(wb.Sheets.Sheet) > 0 {
				ridToTarget := map[string]string{}
				for _, r := range rels.Rel {
					ridToTarget[r.ID] = r.Target
				}
				tgt := ridToTarget[wb.Sheets.Sheet[0].RID]
				if tgt != "" {
					cand := tgt
					if !strings.HasPrefix(tgt, "xl/") {
						cand = "xl/" + strings.TrimPrefix(tgt, "/")
					}
					if _, ok := files[cand]; ok {
						sheetName = cand
					}
				}
			}
		}
	}
	if sheetName == "" {
		for _, n := range names {
			if strings.HasPrefix(n, "xl/worksheets/sheet") {
				sheetName = n
				break
			}
		}
	}
	if sheetName == "" {
		return nil, nil, fmt.Errorf("xlsx has no worksheet")
	}
	var sheet xlsxSheet
	if err := readXML(sheetName, &sheet); err != nil {
		return nil, nil, err
	}
	var grid []map[string]string
	for _, row := range sheet.Rows {
		cells := map[string]string{}
		for _, cell := range row.Cs {
			col := ""
			for _, ch := range cell.R {
				if ch >= 'A' && ch <= 'Z' {
					col += string(ch)
				}
			}
			val := ""
			switch cell.T {
			case "inlineStr":
				var sb strings.Builder
				for _, t := range cell.IS.Ts {
					sb.WriteString(t)
				}
				for _, r := range cell.IS.R {
					sb.WriteString(r.T)
				}
				val = sb.String()
			case "s":
				if idx, err := strconv.Atoi(cell.V); err == nil && idx >= 0 && idx < len(shared) {
					val = shared[idx]
				}
			default:
				val = cell.V
			}
			cells[col] = val
		}
		grid = append(grid, cells)
	}
	if len(grid) == 0 {
		return []string{}, nil, nil
	}
	colSet := map[string]bool{}
	for _, row := range grid {
		for k := range row {
			colSet[k] = true
		}
	}
	cols := make([]string, 0, len(colSet))
	for k := range colSet {
		cols = append(cols, k)
	}
	sort.Slice(cols, func(i, j int) bool { return excelColKey(cols[i]) < excelColKey(cols[j]) })
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = grid[0][c]
	}
	var rows []map[string]string
	for _, row := range grid[1:] {
		r := map[string]string{}
		for i, c := range cols {
			if headers[i] != "" {
				r[headers[i]] = row[c]
			}
		}
		rows = append(rows, r)
	}
	return headers, rows, nil
}

func excelColKey(ref string) int {
	n := 0
	for _, ch := range ref {
		n = n*26 + int(ch) - 64
	}
	return n
}

// --- mapping + normalization -----------------------------------------------------

func proposeMapping(headers []string) map[string]string {
	mapping := map[string]string{}
	lower := map[string]string{}
	for _, h := range headers {
		if h != "" {
			lower[strings.ToLower(strings.TrimSpace(h))] = h
		}
	}
	for _, field := range leadSynonymOrder {
		for _, syn := range leadSynonyms[field] {
			if orig, ok := lower[syn]; ok {
				mapping[field] = orig
				break
			}
		}
	}
	return mapping
}

// pyDumpsSortKeys reproduces json.dumps(mapping, sort_keys=True) byte-for-byte
// (ensure_ascii=True, ", "/": " separators) — the idempotency hash depends on it.
func pyDumpsSortKeys(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(pyJSONString(k))
		sb.WriteString(": ")
		sb.WriteString(pyJSONString(m[k]))
	}
	sb.WriteString("}")
	return sb.String()
}

// pyJSONString escapes like Python json (ensure_ascii=True).
func pyJSONString(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			if r < 0x20 || r > 0x7E {
				if r > 0xFFFF {
					r1, r2 := utf16Pair(r)
					fmt.Fprintf(&sb, `\u%04x\u%04x`, r1, r2)
				} else {
					fmt.Fprintf(&sb, `\u%04x`, r)
				}
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func utf16Pair(r rune) (rune, rune) {
	r -= 0x10000
	return 0xD800 + (r >> 10), 0xDC00 + (r & 0x3FF)
}

func leadIdempotencyKey(path string, mapping map[string]string) (string, error) {
	h := sha256.New()
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	h.Write([]byte(pyDumpsSortKeys(mapping)))
	return fmt.Sprintf("%x", h.Sum(nil))[:16], nil
}

func normalizeLeadRow(raw map[string]string, mapping map[string]string) map[string]any {
	g := func(field string) string {
		col, ok := mapping[field]
		if !ok || col == "" {
			return ""
		}
		return strings.TrimSpace(raw[col])
	}
	full := g("full_name")
	if full == "" {
		var parts []string
		for _, x := range []string{g("first_name"), g("last_name")} {
			if x != "" {
				parts = append(parts, x)
			}
		}
		full = strings.Join(parts, " ")
	}
	socials := map[string]any{}
	for _, s := range []string{"facebook", "linkedin", "instagram"} {
		if v := g(s); v != "" {
			socials[s] = v
		}
	}
	return map[string]any{
		"full_name": full, "first_name": g("first_name"), "last_name": g("last_name"),
		"email": normalizeEmail(g("email")), "phone": normalizePhone(g("phone")),
		"company": g("company"), "website": g("website"),
		"city": g("city"), "state": g("state"), "socials": socials,
	}
}

func leadToContactFields(norm map[string]any) map[string]any {
	emails := []any{}
	if mStr(norm, "email") != "" {
		emails = append(emails, map[string]any{"address": norm["email"], "source": "import",
			"status": "unverified", "is_primary": true})
	}
	phones := []any{}
	if mStr(norm, "phone") != "" {
		phones = append(phones, map[string]any{"number": norm["phone"], "type": "cell", "source": "import"})
	}
	var website any
	if mStr(norm, "website") != "" {
		website = norm["website"]
	}
	fields := map[string]any{
		"name": map[string]any{"full": norm["full_name"], "first": norm["first_name"], "last": norm["last_name"]},
		"identities": map[string]any{"emails": emails, "phones": phones,
			"socials": mMap(norm, "socials"), "website": website},
	}
	custom := map[string]any{}
	for _, k := range []string{"company", "city", "state"} {
		if mStr(norm, k) != "" {
			custom[k] = norm[k]
		}
	}
	fields["custom_fields"] = custom
	if mStr(norm, "email") != "" {
		fields["channels"] = map[string]any{"email": map[string]any{"status": "usable"}}
	}
	return fields
}

// --- import --------------------------------------------------------------------

func doLeadImport(clientDir, file, listSlug string, mapping map[string]string, mxCheck bool) (map[string]any, error) {
	headers, next, closer, err := leadRows(file)
	if err != nil {
		return nil, err
	}
	defer closer()
	if mapping == nil {
		mapping = proposeMapping(headers)
	}
	if len(mapping) == 0 {
		return nil, fmt.Errorf("could not infer a column mapping; pass --mapping explicitly")
	}
	store := newCrmStore(clientDir)
	listDir := filepath.Join(clientDir, "lists", listSlug)
	if err := os.MkdirAll(listDir, 0o755); err != nil {
		return nil, err
	}
	leadsPath := filepath.Join(listDir, "leads.jsonl")
	manifestPath := filepath.Join(listDir, "list_manifest.json")
	idem, err := leadIdempotencyKey(file, mapping)
	if err != nil {
		return nil, err
	}
	if prev, err := readJSONFile(manifestPath); err == nil {
		if mStr(prev, "idempotency_key") == idem {
			return map[string]any{"skipped": true,
				"reason": "already imported (idempotency_key match)", "manifest": prev}, nil
		}
	}
	created, matched, suppressed, skipped, errored := 0, 0, 0, 0, 0
	seq := 0
	lf, err := os.OpenFile(leadsPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	defer lf.Close()
	writeLine := func(m map[string]any) {
		lf.WriteString(marshalLineJSON(m) + "\n")
	}
	for {
		raw, ok := next()
		if !ok {
			break
		}
		seq++
		norm := normalizeLeadRow(raw, mapping)
		if mStr(norm, "email") == "" && mStr(norm, "phone") == "" &&
			len(mMap(norm, "socials")) == 0 && mStr(norm, "full_name") == "" {
			skipped++
			writeLine(map[string]any{"seq": seq, "ts": nowISO(), "normalized": norm,
				"outcome": "skipped_invalid", "lead_id": nil, "reason": "no identity or name"})
			continue
		}
		var socialVals []string
		for _, v := range mMap(norm, "socials") {
			if s, ok := v.(string); ok {
				socialVals = append(socialVals, s)
			}
		}
		if supp := store.isSuppressed(mStr(norm, "email"), mStr(norm, "phone"), socialVals); supp != nil {
			suppressed++
			writeLine(map[string]any{"seq": seq, "ts": nowISO(), "normalized": norm,
				"outcome": "suppressed", "lead_id": nil, "reason": supp["reason"]})
			continue
		}
		if mxCheck && mStr(norm, "email") != "" {
			v := emailCheck(mStr(norm, "email"))
			if !v.MXOK {
				norm["_email_status"] = "email_not_found"
			}
		}
		fields := leadToContactFields(norm)
		if mStr(norm, "email") != "" && mStr(norm, "_email_status") != "" {
			if ems := mList(mMap(fields, "identities"), "emails"); len(ems) > 0 {
				if em, ok := ems[0].(map[string]any); ok {
					em["status"] = norm["_email_status"]
				}
			}
		}
		leadID, outcome, err := store.addContact(fields)
		if err != nil {
			errored++
			writeLine(map[string]any{"seq": seq, "ts": nowISO(), "outcome": "error",
				"lead_id": nil, "reason": fmt.Sprintf("StorageError: %s", err.Error())})
			continue
		}
		if outcome == "created" {
			created++
			if _, err := store.logActivity("imported", leadID, "imported from "+listSlug, "agent",
				nil, map[string]any{"path": "lists/" + listSlug}); err != nil {
				return nil, err
			}
		} else {
			matched++
		}
		writeLine(map[string]any{"seq": seq, "ts": nowISO(), "normalized": norm,
			"outcome": outcome, "lead_id": leadID, "reason": ""})
	}
	absFile, err := filepath.Abs(file)
	if err != nil {
		absFile = file
	}
	manifest := map[string]any{
		"schema_version": 1, "list_slug": listSlug, "source_file": absFile,
		"source_format": strings.TrimPrefix(filepath.Ext(file), "."), "imported_at": nowISO(),
		"idempotency_key": idem, "column_mapping": stringMapAny(mapping), "row_count": seq,
		"contacts_created": created, "contacts_matched_existing": matched,
		"suppressed_at_import": suppressed, "rows_skipped": skipped, "rows_errored": errored, "notes": "",
	}
	if err := os.WriteFile(manifestPath, []byte(marshalIndentJSON(manifest)), 0o644); err != nil {
		return nil, err
	}
	if err := appendImportLog(listDir, manifest); err != nil {
		return nil, err
	}
	return map[string]any{"skipped": false, "manifest": manifest}, nil
}

func stringMapAny(m map[string]string) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func appendImportLog(listDir string, m map[string]any) error {
	path := filepath.Join(listDir, "import_log.md")
	_, statErr := os.Stat(path)
	isNew := statErr != nil
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if isNew {
		if _, err := f.WriteString("# Import Log\n\n| Date | Source | Rows | Created | Matched | Suppressed | Skipped | Blocker |\n|---|---|---|---|---|---|---|---|\n"); err != nil {
			return err
		}
	}
	blocker := "—"
	if mInt(m, "rows_skipped", 0) != 0 {
		blocker = "see leads.jsonl"
	}
	_, err = f.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |\n",
		mStr(m, "imported_at"), filepath.Base(mStr(m, "source_file")), pyNum(m["row_count"]),
		pyNum(m["contacts_created"]), pyNum(m["contacts_matched_existing"]),
		pyNum(m["suppressed_at_import"]), pyNum(m["rows_skipped"]), blocker))
	return err
}

// --- CLI: tool import-leads -------------------------------------------------------

func runImportLeadsCLI(args []string) int {
	valueFlags := map[string]bool{"--file": true, "--rows": true, "--client-dir": true,
		"--list-slug": true, "--mapping": true}
	boolFlags := map[string]bool{"--no-mx-check": true}
	a, err := parseCLIArgs(args, valueFlags, boolFlags)
	if err != nil {
		return crmUsageErr(err.Error())
	}
	if len(a.pos) == 0 {
		return crmUsageErr("a subcommand is required (inspect | import)")
	}
	switch a.pos[0] {
	case "inspect":
		file := a.get("--file")
		if file == "" {
			return crmUsageErr("inspect needs --file")
		}
		headers, next, closer, err := leadRows(file)
		if err != nil {
			return crmFail(err)
		}
		defer closer()
		var rows []any
		total := 0
		nRows := a.getInt("--rows", 5)
		for {
			r, ok := next()
			if !ok {
				break
			}
			total++
			if len(rows) < nRows {
				rows = append(rows, stringMapAny(r))
			}
		}
		if headers == nil {
			headers = []string{}
		}
		return crmOut(map[string]any{"headers": headers, "proposed_mapping": stringMapAny(proposeMapping(headers)),
			"sample_rows": orEmptyList(rows), "total_rows": total,
			"note": "Confirm/adjust the mapping, then run: import_leads.py import --mapping '<json>'"}, 0)
	case "import":
		clientDir, file, listSlug := a.get("--client-dir"), a.get("--file"), a.get("--list-slug")
		if clientDir == "" || file == "" || listSlug == "" {
			return crmUsageErr("import needs --client-dir, --file, --list-slug")
		}
		var mapping map[string]string
		if s := a.get("--mapping"); s != "" {
			var m map[string]any
			if err := json.Unmarshal([]byte(s), &m); err != nil {
				return crmUsageErr("bad --mapping: " + err.Error())
			}
			mapping = map[string]string{}
			for k, v := range m {
				if sv, ok := v.(string); ok {
					mapping[k] = sv
				}
			}
		}
		res, err := doLeadImport(clientDir, file, listSlug, mapping, !a.bools["--no-mx-check"])
		if err != nil {
			return crmFail(err)
		}
		return crmOut(res, 0)
	}
	return crmOut(map[string]any{"error": "unknown command"}, 2)
}
