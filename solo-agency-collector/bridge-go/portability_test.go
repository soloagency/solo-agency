package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportImportRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src", "daily-content-pipeline")
	dst := filepath.Join(tmp, "dst", "daily-content-pipeline")
	srcInstall := filepath.Dir(src) // <tmp>/src
	dstInstall := filepath.Dir(dst) // <tmp>/dst

	mk := func(root, rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	// data
	mk(src, "clients/leadup/main/outreach/crm/contacts/c_1.json", `{"id":"c_1","identities":{"emails":[{"address":"a@x.com"}]}}`)
	mk(src, "clients/leadup/main/outreach/campaigns/camp/campaign_config.json", `{"campaign_slug":"camp","status":"active"}`)
	mk(src, "clients/leadup/main/outreach/campaigns/camp/sent/2026-07/sent_log.jsonl", `{"lead_id":"c_1","rfc_message_id":"<m@x>"}`+"\n")
	mk(src, "clients/leadup/main/outreach/crm/contact_identities.jsonl", `{"kind":"email","value":"a@x.com","contact_id":"c_1","removed":false}`+"\n")
	mk(src, "clients/leadup/main/client_profile_leadup_main.md", "# LeadUp")
	// secrets
	mk(src, "clients/leadup/main/outreach/sendboxes/sb-a/credentials.json", `{"email":"a@gmail.com","app_password":"SECRETPASS"}`)
	mk(src, "clients/leadup/main/integrations/providers/provider_config.local.json", `{"providers":{"widecast":{"api_key_local":"wc_live_SECRET"}}}`)
	mk(src, "bridge/ui_token", "TOKENSECRET")
	// taskdef with a source-absolute path (must rebase)
	mk(src, "schedule.md", "task_name: LeadUp Daily\ntimezone: America/Los_Angeles\ntarget_client_slug: leadup\ncampaign_slug: camp\nnotification_channel: widecast\nprompt_file: daily-content-pipeline/automation/camp_scheduled_run_prompt.md\n")
	mk(src, "automation/camp_scheduled_run_prompt.md", "Runtime state root: "+src+"\nOutreach client directory: "+src+"/clients/leadup/main/outreach\n")
	// shared + junk
	mk(src, "clients_index.md", "leadup")
	mk(src, "provider_defaults.json", `{"notification":{}}`)
	mk(src, "clients/leadup/main/.DS_Store", "junk")
	mk(src, "clients/leadup/main/outreach/crm_20260719_2244.backup/x.json", "snapshot junk")
	mk(src, "clients/leadup/main/client_profile_leadup_main_20260719_1926.md", "snapshot sibling")

	pass := []byte("correct horse battery staple")
	bundle := filepath.Join(tmp, "leadup.sagx.zip")

	// EXPORT (agency)
	er, err := exportBundle(src, "agency", nil, bundle, pass, "claude", false)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if er["secret_files"].(int) != 3 {
		t.Fatalf("want 3 secret files, got %v", er["secret_files"])
	}
	if er["tasks"].(int) < 1 {
		t.Fatalf("expected a task parsed from schedule.md, got %v", er["tasks"])
	}

	// IMPORT to fresh dest
	ir, err := importBundle(bundle, dst, pass, false, false)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if ok, _ := ir["ok"].(bool); !ok {
		t.Fatalf("import not ok: %v", ir)
	}
	// data placed
	if b, _ := os.ReadFile(filepath.Join(dst, "clients/leadup/main/outreach/crm/contacts/c_1.json")); !strings.Contains(string(b), "c_1") {
		t.Fatal("data file not placed")
	}
	// junk excluded
	if _, e := os.Stat(filepath.Join(dst, "clients/leadup/main/.DS_Store")); e == nil {
		t.Fatal(".DS_Store must be excluded")
	}
	if _, e := os.Stat(filepath.Join(dst, "clients/leadup/main/outreach/crm_20260719_2244.backup/x.json")); e == nil {
		t.Fatal("external .backup snapshot must be excluded")
	}
	if _, e := os.Stat(filepath.Join(dst, "clients/leadup/main/client_profile_leadup_main_20260719_1926.md")); e == nil {
		t.Fatal("timestamped snapshot sibling must be excluded")
	}
	// secrets decrypted + 0600
	cred := filepath.Join(dst, "clients/leadup/main/outreach/sendboxes/sb-a/credentials.json")
	b, e := os.ReadFile(cred)
	if e != nil || !strings.Contains(string(b), "SECRETPASS") {
		t.Fatalf("secret not restored: %v", e)
	}
	if info, _ := os.Stat(cred); info.Mode().Perm() != 0o600 {
		t.Fatalf("secret must be 0600, got %v", info.Mode())
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "clients/leadup/main/integrations/providers/provider_config.local.json")); !strings.Contains(string(b), "wc_live_SECRET") {
		t.Fatal("workspace-root provider secret not restored")
	}
	// taskdef rebased: source install path replaced by dest install path
	pb, _ := os.ReadFile(filepath.Join(dst, "automation/camp_scheduled_run_prompt.md"))
	if strings.Contains(string(pb), srcInstall) {
		t.Fatalf("source path not rebased: %s", pb)
	}
	if !strings.Contains(string(pb), dstInstall) {
		t.Fatalf("dest path missing after rebase: %s", pb)
	}
	if len(ir["residual_source_paths"].([]string)) != 0 {
		t.Fatalf("residual source paths after rebase: %v", ir["residual_source_paths"])
	}

	// GUARDRAIL: re-import onto populated dest without --force -> refused
	ir2, _ := importBundle(bundle, dst, pass, false, false)
	if ok, _ := ir2["ok"].(bool); ok {
		t.Fatal("import onto populated dest must refuse without --force")
	}

	// WRONG passphrase -> decrypt fails, import not ok
	ir3, _ := importBundle(bundle, filepath.Join(tmp, "dst2", "daily-content-pipeline"), []byte("wrong"), false, false)
	if ok, _ := ir3["ok"].(bool); ok {
		t.Fatal("wrong passphrase must fail")
	}
}

func TestExportNoSecrets(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src", "daily-content-pipeline")
	mk := func(rel, body string) {
		p := filepath.Join(src, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	mk("clients/leadup/main/outreach/crm/contacts/c_1.json", `{"id":"c_1"}`)
	mk("clients/leadup/main/outreach/sendboxes/sb-a/credentials.json", `{"app_password":"x"}`)
	bundle := filepath.Join(tmp, "b.zip")
	// no passphrase, no-secrets -> succeeds, records omitted secrets
	r, err := exportBundle(src, "agency", nil, bundle, nil, "codex", true)
	if err != nil {
		t.Fatalf("no-secrets export: %v", err)
	}
	if r["secrets_encrypted"].(bool) {
		t.Fatal("no-secrets must not encrypt")
	}
	// with secrets but no passphrase -> error
	if _, err := exportBundle(src, "agency", nil, filepath.Join(tmp, "b2.zip"), nil, "codex", false); err == nil {
		t.Fatal("secret export without passphrase must error")
	}
}

func TestPBKDF2KnownVector(t *testing.T) {
	// RFC 6070-style sanity: PBKDF2-HMAC-SHA256, password="password", salt="salt",
	// c=1, dkLen=32 -> known value
	got := pbkdf2SHA256([]byte("password"), []byte("salt"), 1, 32)
	want := "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b"
	if sha256HexRaw(got) != want {
		t.Fatalf("pbkdf2 c=1 mismatch:\n got=%s\nwant=%s", sha256HexRaw(got), want)
	}
}

func sha256HexRaw(b []byte) string {
	const hexd = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = hexd[c>>4]
		out[i*2+1] = hexd[c&0xf]
	}
	return string(out)
}

func TestPortClassifyContract(t *testing.T) {
	secrets := []string{
		"clients/leadup/main/outreach/sendboxes/sb-a/credentials.json",
		"clients/leadup/main/integrations/providers/provider_config.local.json",
		"clients/leadup/main/outreach/tracking/.tenant_secret.json",
		"bridge/ui_token",
		"secrets/oauth_client.json",
	}
	for _, s := range secrets {
		if portClassify(s) != "secret" {
			t.Errorf("%q must classify as secret, got %q", s, portClassify(s))
		}
	}
	if portClassify("schedule.md") != "taskdef" || portClassify("automation/x_scheduled_run_prompt.md") != "taskdef" {
		t.Error("schedule/automation must be taskdef")
	}
	if portClassify("clients_index.md") != "shared" || portClassify("provider_defaults.json") != "shared" {
		t.Error("index/defaults must be shared")
	}
	if portClassify("clients/x/main/outreach/crm/contact_identities.jsonl") != "cursor" {
		t.Error("identity index must be cursor")
	}
	// benign 0600 must NOT be flagged as secret
	if portClassify("collector/inbox/completed_runs.json") == "secret" ||
		!portBenign0600("collector/inbox/completed_runs.json") {
		t.Error("completed_runs.json is benign 0600, not a secret")
	}
	if !portBenign0600("clients/x/main/outreach/ui_inbox/approval_decisions.jsonl") {
		t.Error("ui_inbox events are benign 0600")
	}
}

func TestPortGrowthGuards(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src", "daily-content-pipeline")
	srcInstall := filepath.Dir(src)
	mk := func(rel, body string, perm os.FileMode) {
		p := filepath.Join(src, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
		os.Chmod(p, perm)
	}
	mk("clients/x/main/outreach/crm/contacts/c.json", `{"id":"c"}`, 0o644)
	// GUARD 1: a NEW 0600 file the classifier doesn't know -> flagged (still copied)
	mk("clients/x/main/outreach/vault/new_api_token.json", `{"token":"xyz"}`, 0o600)
	// benign 0600 -> NOT flagged
	mk("collector/inbox/completed_runs.json", `[]`, 0o600)
	// GUARD 2: a non-taskdef data file carrying the source path -> reported on import
	mk("clients/x/main/outreach/notes.md", "see "+src+"/clients/x for details", 0o644)

	bundle := filepath.Join(tmp, "b.zip")
	pass := []byte("pw")
	res, err := exportBundle(src, "agency", nil, bundle, pass, "claude", false)
	if err != nil {
		t.Fatal(err)
	}
	// guard 1: the unknown 0600 file surfaced, benign one did not
	us, _ := res["unclassified_sensitive"].([]string)
	found := false
	for _, u := range us {
		if strings.HasSuffix(u, "new_api_token.json") {
			found = true
		}
		if strings.HasSuffix(u, "completed_runs.json") {
			t.Fatal("benign 0600 must not be flagged")
		}
	}
	if !found {
		t.Fatalf("new 0600 file not flagged as unclassified_sensitive: %v", us)
	}
	_ = srcInstall

	// guard 2: import reports the residual source path in the non-taskdef file
	dst := filepath.Join(tmp, "dst", "daily-content-pipeline")
	ir, err := importBundle(bundle, dst, pass, false, false)
	if err != nil {
		t.Fatal(err)
	}
	resid, _ := ir["residual_source_paths"].([]string)
	hit := false
	for _, r := range resid {
		if strings.HasSuffix(r, "notes.md") {
			hit = true
		}
	}
	if !hit {
		t.Fatalf("residual source path in non-taskdef file not reported: %v", resid)
	}
}
