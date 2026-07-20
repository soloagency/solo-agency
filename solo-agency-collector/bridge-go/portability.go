package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// portability.go — export/import the WORKING ENVIRONMENT (data + configure +
// task manifest) so a client or the whole agency can move between machines and
// between AI agents (Claude <-> Codex) with the environment restored intact.
//
// Four-layer model (see docs + WIDECAST_TRACKING_SPEC-style contract):
//   - DATA    : CRM, campaigns, sent logs, reports, profiles, cursors — copied
//               verbatim (cursors especially: moving them as-is is what keeps
//               idempotency, i.e. no double-send / no re-notify after a move).
//   - SECRET  : Gmail App Passwords, WideCast key, tracking secret, ui_token —
//               packed into an AES-256-GCM blob, key = PBKDF2(passphrase). The
//               agent never sees the passphrase (operator supplies it via env)
//               nor the plaintext secrets.
//   - TASKDEF : automation/schedule prose (no canonical schema in this system)
//               carried + path-rebased; the DESTINATION agent re-registers the
//               tasks in ITS own scheduler (the one cross-agent seam).
//   - EXCLUDE : bridge binary, chrome extension, autostart registration, PID/log
//               — machine-specific, rebuilt on the destination by setup_collector.
//
// The bundle is a .zip: manifest.json + data/<rel...> + secrets.enc.

const portSchemaVersion = 1
const portPBKDF2Iters = 210000
const portPassphraseEnv = "SOLO_AGENCY_EXPORT_PASSPHRASE"

type portFile struct {
	Rel   string `json:"rel"`
	SHA   string `json:"sha256"`
	Size  int64  `json:"size"`
	Class string `json:"class"` // data | cursor | taskdef | config | shared
}

type portTask struct {
	Name         string `json:"name,omitempty"`
	Cadence      string `json:"cadence,omitempty"`
	Timezone     string `json:"timezone,omitempty"`
	ClientSlug   string `json:"client_slug,omitempty"`
	CampaignSlug string `json:"campaign_slug,omitempty"`
	Notification string `json:"notification_channel,omitempty"`
	PromptFile   string `json:"prompt_file,omitempty"` // rel path to the run-prompt prose
	NativeID     string `json:"native_automation_id,omitempty"`
}

type portManifest struct {
	SchemaVersion    int        `json:"schema_version"`
	CreatedAt        string     `json:"created_at"`
	SourceAgent      string     `json:"source_agent"`
	Scope            string     `json:"scope"` // agency | client
	Clients          []string   `json:"clients"`
	SourceDataRoot   string     `json:"source_data_root"`
	SourceInstall    string     `json:"source_install_root"`
	ToolVersion      string     `json:"tool_version"`
	Files            []portFile `json:"files"`
	Tasks            []portTask `json:"tasks"`
	SecretsEncrypted bool       `json:"secrets_encrypted"`
	SecretFiles      []string   `json:"secret_files"`
	SecretKDF        string     `json:"secret_kdf,omitempty"`
	SecretIters      int        `json:"secret_iters,omitempty"`
}

// ---------- classification ----------

var portTimestampBackupRe = regexp.MustCompile(`_[0-9]{8}_[0-9]{4}\.backup(/|$)`)

// external snapshot-wrapper siblings: file_YYYYMMDD_HHMM.ext (this system's own
// data never uses that suffix — it partitions by YYYY-MM/ or YYYY-MM-DD/ folders)
var portSnapshotSiblingRe = regexp.MustCompile(`_[0-9]{8}_[0-9]{4}\.[A-Za-z0-9]+$`)

// portExcluded reports paths that must NOT travel (machine junk, external
// snapshot wrapper artifacts, self-healing heartbeat).
func portExcluded(rel string) bool {
	base := filepath.Base(rel)
	switch base {
	case ".DS_Store", "bridge_health.json":
		return true
	}
	if strings.HasSuffix(rel, ".backup") || portTimestampBackupRe.MatchString(rel) || portSnapshotSiblingRe.MatchString(base) {
		return true // external snapshot-wrapper twins, not our schema
	}
	if strings.HasPrefix(rel, "automation/backups/") {
		return true // update-watch runtime rollback snapshots — dest has its own
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".git" || seg == "node_modules" {
			return true
		}
		if strings.HasSuffix(seg, ".backup") {
			return true
		}
	}
	return false
}

// portClassify returns the handling class for a data-root-relative path.
// "secret" is returned separately (goes into the encrypted blob, never plain).
func portClassify(rel string) string {
	base := filepath.Base(rel)
	// secrets
	if base == "credentials.json" || base == "ui_token" || base == ".tenant_secret.json" ||
		strings.HasPrefix(base, "provider_config.local") {
		return "secret"
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == "secrets" {
			return "secret"
		}
	}
	// cursors / idempotency (data, but flagged so import never resets them)
	if base == "sendboxes.json" || base == "sent_log.jsonl" || base == ".reply_notify_seen.json" ||
		base == ".pull_cursor.json" || base == ".approval_cursor" || base == "contact_identities.jsonl" ||
		base == "completed_runs.json" || strings.HasPrefix(rel, "clients/") && strings.Contains(rel, "/_reservations/") {
		return "cursor"
	}
	// task definitions (prose, path-rebased on import)
	if rel == "schedule.md" || strings.HasPrefix(rel, "automation/") {
		return "taskdef"
	}
	// agency-shared files: placed only-if-absent on a single-client import
	if rel == "clients_index.md" || rel == "provider_defaults.json" {
		return "shared"
	}
	return "data"
}

// ---------- crypto (stdlib only) ----------

func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	h := sha256.New
	hashLen := sha256.Size
	blocks := (keyLen + hashLen - 1) / hashLen
	var dk []byte
	buf := make([]byte, 4)
	for block := 1; block <= blocks; block++ {
		buf[0], buf[1], buf[2], buf[3] = byte(block>>24), byte(block>>16), byte(block>>8), byte(block)
		prf := hmac.New(h, password)
		prf.Write(salt)
		prf.Write(buf)
		u := prf.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for i := range t {
				t[i] ^= u[i]
			}
		}
		dk = append(dk, t...)
	}
	return dk[:keyLen]
}

// portEncrypt: secrets.enc = salt(16) || nonce(12) || AES-256-GCM(ciphertext).
func portEncrypt(plaintext, passphrase []byte) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key := pbkdf2SHA256(passphrase, salt, portPBKDF2Iters, 32)
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, salt) // salt as AAD binds the KDF salt
	out := append(append(append([]byte{}, salt...), nonce...), ct...)
	return out, nil
}

func portDecrypt(blob, passphrase []byte) ([]byte, error) {
	if len(blob) < 16+12 {
		return nil, fmt.Errorf("secrets blob too short")
	}
	salt, rest := blob[:16], blob[16:]
	key := pbkdf2SHA256(passphrase, salt, portPBKDF2Iters, 32)
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	nonce, ct := rest[:ns], rest[ns:]
	pt, err := gcm.Open(nil, nonce, ct, salt)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed (wrong passphrase or corrupt bundle)")
	}
	return pt, nil
}

// ---------- collection ----------

func sha256Hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// portInScope reports whether a data-root-relative path belongs to this export.
func portInScope(rel, scope string, clientSet map[string]bool, campaignPromptSet map[string]bool) bool {
	if scope == "agency" {
		return true
	}
	// client scope: the client workspace subtree(s) + that client's task prose
	if strings.HasPrefix(rel, "clients/") {
		parts := strings.SplitN(rel, "/", 3)
		if len(parts) >= 2 && clientSet[parts[1]] {
			return true
		}
		return false
	}
	if rel == "provider_defaults.json" {
		return true // shared default, placed only-if-absent on import
	}
	if strings.HasPrefix(rel, "automation/") {
		return campaignPromptSet[filepath.Base(rel)] // only this client's prompt files
	}
	return false
}

// ---------- export ----------

func exportBundle(dataRoot, scope string, clients []string, outPath string, passphrase []byte, sourceAgent string, noSecrets bool) (map[string]any, error) {
	dataRoot = filepath.Clean(dataRoot)
	if scope != "agency" && scope != "client" {
		return nil, fmt.Errorf("scope must be agency|client")
	}
	clientSet := map[string]bool{}
	for _, c := range clients {
		clientSet[c] = true
	}
	// campaigns of the selected clients -> their automation prompt filenames
	campaignPromptSet := map[string]bool{}
	if scope == "client" {
		for _, c := range clients {
			base := filepath.Join(dataRoot, "clients", c)
			wss, _ := os.ReadDir(base)
			for _, ws := range wss {
				camps, _ := os.ReadDir(filepath.Join(base, ws.Name(), "outreach", "campaigns"))
				for _, cm := range camps {
					campaignPromptSet[cm.Name()+"_scheduled_run_prompt.md"] = true
				}
			}
		}
	}

	man := portManifest{
		SchemaVersion: portSchemaVersion, CreatedAt: nowISO(), SourceAgent: strOr(sourceAgent, "unknown"),
		Scope: scope, Clients: clients, SourceDataRoot: dataRoot, SourceInstall: filepath.Dir(dataRoot),
		ToolVersion: "bridge-go", Tasks: parseTasks(dataRoot, scope, clientSet),
	}

	var dataFiles []struct{ rel, abs string } // non-secret
	var secretFiles []string
	err := filepath.WalkDir(dataRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(dataRoot, p)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if portExcluded(rel) || !portInScope(rel, scope, clientSet, campaignPromptSet) {
			return nil
		}
		if portClassify(rel) == "secret" {
			secretFiles = append(secretFiles, rel)
		} else {
			dataFiles = append(dataFiles, struct{ rel, abs string }{rel, p})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(dataFiles, func(i, j int) bool { return dataFiles[i].rel < dataFiles[j].rel })
	sort.Strings(secretFiles)
	// distinct client slugs present (agency scope carries no explicit list) so
	// the import clash-check + reindex know which subtrees moved
	{
		allRels := make([]portFile, 0, len(dataFiles))
		for _, f := range dataFiles {
			allRels = append(allRels, portFile{Rel: f.rel})
		}
		man.Clients = portClientsFromFiles(allRels, secretFiles)
	}

	// build the zip
	tmp := outPath + ".part"
	zf, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}
	zw := zip.NewWriter(zf)

	for _, f := range dataFiles {
		b, rerr := os.ReadFile(f.abs)
		if rerr != nil {
			continue
		}
		man.Files = append(man.Files, portFile{Rel: f.rel, SHA: sha256Hex(b), Size: int64(len(b)), Class: portClassify(f.rel)})
		w, werr := zw.Create("data/" + f.rel)
		if werr != nil {
			zw.Close()
			zf.Close()
			os.Remove(tmp)
			return nil, werr
		}
		w.Write(b)
	}

	// secrets -> tar -> encrypt -> secrets.enc  (unless --no-secrets: record the
	// omitted paths so import can tell the operator to re-auth them)
	if noSecrets {
		man.SecretsEncrypted = false
		man.SecretFiles = secretFiles
		secretFiles = nil
	}
	if len(secretFiles) > 0 {
		if len(passphrase) == 0 {
			zw.Close()
			zf.Close()
			os.Remove(tmp)
			return nil, fmt.Errorf("this export contains %d secret file(s); set %s (the operator's passphrase) or pass --no-secrets", len(secretFiles), portPassphraseEnv)
		}
		var tarBuf bytes.Buffer
		tw := tar.NewWriter(&tarBuf)
		for _, rel := range secretFiles {
			b, rerr := os.ReadFile(filepath.Join(dataRoot, filepath.FromSlash(rel)))
			if rerr != nil {
				continue
			}
			tw.WriteHeader(&tar.Header{Name: rel, Mode: 0o600, Size: int64(len(b))})
			tw.Write(b)
		}
		tw.Close()
		enc, eerr := portEncrypt(tarBuf.Bytes(), passphrase)
		if eerr != nil {
			zw.Close()
			zf.Close()
			os.Remove(tmp)
			return nil, eerr
		}
		w, _ := zw.Create("secrets.enc")
		w.Write(enc)
		man.SecretsEncrypted = true
		man.SecretFiles = secretFiles
		man.SecretKDF = "pbkdf2-hmac-sha256"
		man.SecretIters = portPBKDF2Iters
	}

	mw, _ := zw.Create("manifest.json")
	mw.Write([]byte(marshalIndentJSON(portManifestToMap(man))))
	if err := zw.Close(); err != nil {
		zf.Close()
		os.Remove(tmp)
		return nil, err
	}
	if err := zf.Close(); err != nil {
		os.Remove(tmp)
		return nil, err
	}
	if err := os.Rename(tmp, outPath); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "out": outPath, "scope": scope, "clients": man.Clients,
		"data_files": len(man.Files), "secret_files": len(man.SecretFiles), "tasks": len(man.Tasks),
		"secrets_encrypted": man.SecretsEncrypted}, nil
}

// portClientsFromFiles derives the distinct client slugs present in the bundle
// (from clients/{slug}/... paths) so the manifest carries them regardless of scope.
func portClientsFromFiles(files []portFile, secretRels []string) []string {
	seen := map[string]bool{}
	add := func(rel string) {
		if strings.HasPrefix(rel, "clients/") {
			if parts := strings.SplitN(rel, "/", 3); len(parts) >= 2 && parts[1] != "" {
				seen[parts[1]] = true
			}
		}
	}
	for _, f := range files {
		add(f.Rel)
	}
	for _, r := range secretRels {
		add(r)
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func portManifestToMap(m portManifest) map[string]any {
	b, _ := json.Marshal(m)
	var out map[string]any
	json.Unmarshal(b, &out)
	return out
}

// ---------- task parse (best-effort from prose schedule.md) ----------

var portKVRe = regexp.MustCompile(`(?m)^\s*[-*]?\s*` + "`?" + `([a-z_]+)` + "`?" + `\s*[:=]\s*(.+?)\s*$`)

// parseTasks pulls what structure it can from schedule.md (this system keeps no
// canonical task schema — the destination agent re-registers from these hints +
// the carried prompt files).
func parseTasks(dataRoot, scope string, clientSet map[string]bool) []portTask {
	raw, err := os.ReadFile(filepath.Join(dataRoot, "schedule.md"))
	if err != nil {
		return nil
	}
	var tasks []portTask
	var cur portTask
	flush := func() {
		if cur.Name != "" || cur.CampaignSlug != "" || cur.PromptFile != "" {
			if scope == "agency" || cur.ClientSlug == "" || clientSet[cur.ClientSlug] {
				tasks = append(tasks, cur)
			}
		}
		cur = portTask{}
	}
	globalTZ := ""
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") || strings.TrimSpace(line) == "---" {
			flush()
			continue
		}
		m := portKVRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		k, v := m[1], strings.Trim(m[2], "`\"' ")
		switch k {
		case "task_name", "client_task_name", "campaign_name", "scheduler_name":
			if cur.Name == "" {
				cur.Name = v
			}
		case "cadence", "run_time", "run_window":
			cur.Cadence = strings.TrimSpace(cur.Cadence + " " + v)
		case "timezone":
			cur.Timezone = v
			if globalTZ == "" {
				globalTZ = v
			}
		case "target_client_slug", "active_client":
			cur.ClientSlug = v
		case "campaign_slug":
			cur.CampaignSlug = v
		case "notification_channel":
			cur.Notification = v
		case "prompt_file":
			cur.PromptFile = filepath.ToSlash(v)
		case "native_automation_id":
			cur.NativeID = v
		}
	}
	flush()
	for i := range tasks {
		if tasks[i].Timezone == "" {
			tasks[i].Timezone = globalTZ
		}
	}
	return tasks
}

// ---------- import ----------

func importBundle(bundlePath, destRoot string, passphrase []byte, dryRun, force bool) (map[string]any, error) {
	destRoot = filepath.Clean(destRoot)
	zr, err := zip.OpenReader(bundlePath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	var man portManifest
	byName := map[string]*zip.File{}
	for _, f := range zr.File {
		byName[f.Name] = f
	}
	mf := byName["manifest.json"]
	if mf == nil {
		return nil, fmt.Errorf("not a Solo Agency bundle: manifest.json missing")
	}
	if err := readZipJSON(mf, &man); err != nil {
		return nil, err
	}
	if man.SchemaVersion > portSchemaVersion {
		return nil, fmt.Errorf("bundle schema v%d is newer than this bridge (v%d); update the destination first", man.SchemaVersion, portSchemaVersion)
	}

	// guardrail: never silently overwrite a populated destination
	clash := []string{}
	for _, c := range man.Clients {
		if st, e := os.Stat(filepath.Join(destRoot, "clients", c)); e == nil && st.IsDir() {
			clash = append(clash, c)
		}
	}
	report := map[string]any{
		"scope": man.Scope, "clients": man.Clients, "source_agent": man.SourceAgent,
		"data_files": len(man.Files), "secret_files": len(man.SecretFiles), "tasks": man.Tasks,
		"dest_root": destRoot, "dry_run": dryRun,
	}
	if len(clash) > 0 {
		report["conflict_existing_clients"] = clash
		if !force {
			report["ok"] = false
			report["error"] = "destination already has these client(s); re-run with --force to overwrite (a MOVE should deactivate the source first — see the migration playbook)"
			return report, nil
		}
	}
	// rebase map: replace the source install-root prefix with the destination's
	srcInstall := man.SourceInstall
	dstInstall := filepath.Dir(destRoot)
	rebased := map[string]int{}

	place := func(rel string, content []byte, cls string) error {
		abs := filepath.Join(destRoot, filepath.FromSlash(rel))
		if cls == "shared" {
			if _, e := os.Stat(abs); e == nil {
				return nil // agency-shared file already present on dest — don't clobber
			}
		}
		if cls == "taskdef" && srcInstall != "" && dstInstall != "" && srcInstall != dstInstall {
			if n := bytes.Count(content, []byte(srcInstall)); n > 0 {
				content = bytes.ReplaceAll(content, []byte(srcInstall), []byte(dstInstall))
				rebased[rel] = n
			}
		}
		if dryRun {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return err
		}
		return os.WriteFile(abs, content, 0o644)
	}

	// 1. data (+ taskdef rebase, shared only-if-absent)
	placed := 0
	for _, fm := range man.Files {
		zf := byName["data/"+fm.Rel]
		if zf == nil {
			continue
		}
		b, e := readZipBytes(zf)
		if e != nil {
			return nil, e
		}
		if sha256Hex(b) != fm.SHA {
			return nil, fmt.Errorf("checksum mismatch on %s (corrupt bundle)", fm.Rel)
		}
		if err := place(fm.Rel, b, fm.Class); err != nil {
			return nil, err
		}
		placed++
	}
	report["placed_data_files"] = placed
	report["rebased_files"] = rebased

	// 2. secrets (decrypt -> untar -> 0600)
	secretsPlaced := 0
	if man.SecretsEncrypted {
		if sf := byName["secrets.enc"]; sf != nil {
			if len(passphrase) == 0 {
				report["ok"] = false
				report["error"] = fmt.Sprintf("bundle holds %d encrypted secret(s); set %s to the passphrase used at export", len(man.SecretFiles), portPassphraseEnv)
				return report, nil
			}
			enc, e := readZipBytes(sf)
			if e != nil {
				return nil, e
			}
			pt, e := portDecrypt(enc, passphrase)
			if e != nil {
				report["ok"] = false
				report["error"] = e.Error()
				return report, nil
			}
			tr := tar.NewReader(bytes.NewReader(pt))
			for {
				hdr, terr := tr.Next()
				if terr == io.EOF {
					break
				}
				if terr != nil {
					return nil, terr
				}
				buf, _ := io.ReadAll(tr)
				if !dryRun {
					abs := filepath.Join(destRoot, filepath.FromSlash(hdr.Name))
					if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
						return nil, err
					}
					if err := os.WriteFile(abs, buf, 0o600); err != nil {
						return nil, err
					}
				}
				secretsPlaced++
			}
		}
	}
	report["placed_secret_files"] = secretsPlaced

	// 3. rebuild identity indexes for the imported clients (safety net)
	rebuilt := []string{}
	if !dryRun {
		for _, c := range man.Clients {
			base := filepath.Join(destRoot, "clients", c)
			wss, _ := os.ReadDir(base)
			for _, ws := range wss {
				od := filepath.Join(base, ws.Name(), "outreach")
				if st, e := os.Stat(filepath.Join(od, "crm")); e == nil && st.IsDir() {
					if _, e := newCrmStore(od).validate(true); e == nil {
						rebuilt = append(rebuilt, c+"/"+ws.Name())
					}
				}
			}
		}
	}
	report["reindexed"] = rebuilt

	// residual source-path scan (anything the rebase missed -> agent must fix)
	report["residual_source_paths"] = portResidualPaths(destRoot, man, dryRun)
	report["ok"] = true
	report["next_steps"] = portNextSteps(man)
	return report, nil
}

func portResidualPaths(destRoot string, man portManifest, dryRun bool) []string {
	if dryRun || man.SourceInstall == "" {
		return nil
	}
	var hits []string
	for _, fm := range man.Files {
		if fm.Class != "taskdef" {
			continue
		}
		abs := filepath.Join(destRoot, filepath.FromSlash(fm.Rel))
		b, err := os.ReadFile(abs)
		if err == nil && bytes.Contains(b, []byte(man.SourceInstall)) {
			hits = append(hits, fm.Rel)
		}
	}
	return hits
}

func portNextSteps(man portManifest) []string {
	steps := []string{
		"Run setup_collector on this machine so the bridge binary + autostart are installed for this OS (the bundle carries data + config, not the binary).",
		"Re-register the automation task(s) below in THIS agent's own scheduler (Claude scheduled tasks / Codex tasks) using the carried prompt file(s); the bundle cannot create another agent's tasks.",
		"Reinstall the Chrome extension for each client and sign into that client's Chrome profile (browser sessions cannot be exported).",
	}
	if len(man.Tasks) > 0 {
		steps = append(steps, fmt.Sprintf("%d task(s) parsed from schedule.md — see the tasks list.", len(man.Tasks)))
	}
	steps = append(steps, "On the SOURCE machine: pause its campaigns + disable its scheduled tasks + stop its bridge, so the same campaign never sends from two machines (a MOVE, not a clone).")
	return steps
}

// ---------- zip read helpers ----------

func readZipBytes(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func readZipJSON(f *zip.File, v any) error {
	b, err := readZipBytes(f)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// ---------- CLI ----------

// runMigrateCLI handles `tool migrate export|import ...`. The passphrase is read
// ONLY from the env var (portPassphraseEnv) so the operator supplies it without
// it landing on the command line / process list, and the agent orchestrating the
// call need never see it.
func runMigrateCLI(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: tool migrate export --data-root R --scope agency|client [--clients a,b] --out FILE [--no-secrets] [--agent NAME]")
		fmt.Fprintln(os.Stderr, "       tool migrate import --file FILE --root DEST [--dry-run] [--force]")
		return 2
	}
	pass := []byte(os.Getenv(portPassphraseEnv))
	switch args[0] {
	case "export":
		dataRoot := flagValue(args, "--data-root")
		scope := strOr(flagValue(args, "--scope"), "agency")
		out := flagValue(args, "--out")
		if dataRoot == "" || out == "" {
			fmt.Fprintln(os.Stderr, "export needs --data-root and --out")
			return 2
		}
		var clients []string
		if cs := flagValue(args, "--clients"); cs != "" {
			for _, c := range strings.Split(cs, ",") {
				if c = strings.TrimSpace(c); c != "" {
					clients = append(clients, c)
				}
			}
		}
		if scope == "client" && len(clients) == 0 {
			fmt.Fprintln(os.Stderr, "client scope needs --clients a,b")
			return 2
		}
		if hasFlag(args, "--no-secrets") {
			pass = nil // signal: skip secrets entirely
		}
		res, err := exportBundle(dataRoot, scope, clients, out, pass, flagValue(args, "--agent"), hasFlag(args, "--no-secrets"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: "+err.Error())
			return 1
		}
		printJSONLine(res)
		return 0
	case "import":
		file := flagValue(args, "--file")
		dest := flagValue(args, "--root")
		if file == "" || dest == "" {
			fmt.Fprintln(os.Stderr, "import needs --file and --root")
			return 2
		}
		res, err := importBundle(file, dest, pass, hasFlag(args, "--dry-run"), hasFlag(args, "--force"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: "+err.Error())
			return 1
		}
		printJSONLine(res)
		if ok, _ := res["ok"].(bool); !ok {
			return 1
		}
		return 0
	}
	fmt.Fprintln(os.Stderr, "unknown migrate subcommand: "+args[0])
	return 2
}

func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}
