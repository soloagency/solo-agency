package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeJSONT(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(v)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSendOperatorPicksHealthySendbox(t *testing.T) {
	pipeline := t.TempDir()
	writeJSONT(t, systemSettingsPath(pipeline), map[string]any{
		"schema_version": 1, "operator_email": "op@example.com"})

	// Client A: only a needs_reauth box — must be skipped.
	aDir := filepath.Join(pipeline, "clients", "a", "biz_loc", "outreach")
	writeJSONT(t, filepath.Join(aDir, "sendboxes", "sendboxes.json"), map[string]any{
		"sendboxes": []any{map[string]any{"slug": "sb-bad", "status": "needs_reauth", "email": "bad@x.com"}}})
	writeJSONT(t, credPath(aDir, "sb-bad"), map[string]any{"email": "bad@x.com", "app_password": "p"})

	// Client B: healthy box with credentials — the pick.
	bDir := filepath.Join(pipeline, "clients", "b", "biz_loc", "outreach")
	writeJSONT(t, filepath.Join(bDir, "sendboxes", "sendboxes.json"), map[string]any{
		"sendboxes": []any{map[string]any{"slug": "sb-good", "status": "healthy", "email": "good@x.com"}}})
	writeJSONT(t, credPath(bDir, "sb-good"), map[string]any{"email": "good@x.com", "app_password": "secret"})

	var gotAuth, gotFrom, gotTo, gotRaw string
	orig := operatorSMTPSend
	operatorSMTPSend = func(authEmail, appPassword, from, to string, raw []byte) error {
		gotAuth, gotFrom, gotTo, gotRaw = authEmail, from, to, string(raw)
		return nil
	}
	defer func() { operatorSMTPSend = orig }()

	code := 0
	out := captureStdout(t, func() {
		code = runGmailCLI([]string{"send-operator", "--pipeline", pipeline,
			"--subject", "Client X im 6 ngày", "--body", "Đã nhắc 3 lần, cần anh gọi trực tiếp."})
	})
	if code != 0 {
		t.Fatalf("send-operator exited %d\n%s", code, out)
	}
	if gotAuth != "good@x.com" || gotFrom != "good@x.com" || gotTo != "op@example.com" {
		t.Fatalf("wrong send params: auth=%s from=%s to=%s", gotAuth, gotFrom, gotTo)
	}
	if !strings.Contains(gotRaw, "Subject: Client X im 6 ngày") || !strings.Contains(gotRaw, "nhắc 3 lần") {
		t.Fatalf("MIME missing content:\n%s", gotRaw)
	}
	// Audit line written.
	raw, err := os.ReadFile(filepath.Join(pipeline, "automation", "operator_mail_log.jsonl"))
	if err != nil || !strings.Contains(string(raw), "sb-good") {
		t.Fatalf("audit log missing: %v %s", err, raw)
	}
}

func TestSendOperatorFailsWithoutEmailOrBox(t *testing.T) {
	pipeline := t.TempDir()
	// No operator email configured.
	code := 0
	_ = captureStdout(t, func() {
		code = runGmailCLI([]string{"send-operator", "--pipeline", pipeline,
			"--subject", "s", "--body", "b"})
	})
	if code == 0 {
		t.Fatal("must fail without operator_email")
	}
	// Email set but no healthy sendbox anywhere.
	writeJSONT(t, systemSettingsPath(pipeline), map[string]any{"operator_email": "op@example.com"})
	_ = captureStdout(t, func() {
		code = runGmailCLI([]string{"send-operator", "--pipeline", pipeline,
			"--subject", "s", "--body", "b"})
	})
	if code == 0 {
		t.Fatal("must fail without a healthy sendbox")
	}
}
