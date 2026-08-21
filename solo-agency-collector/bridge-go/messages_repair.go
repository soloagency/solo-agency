package main

// messages_repair.go — put a readable body back on a row that stored the wire
// form. A build briefly saved mimeMessage.Body (quoted-printable, as SMTP needs
// it) instead of the pre-encoding text, so the conversation view showed
// "=C4=90=E1=BA=A7u" for a message the recipient read perfectly well.
//
// The true text is recoverable exactly by decoding, so this is a repair and not
// a reconstruction. Nothing is rewritten: the correction is a NEW row carrying
// supersedes, and the reader already prefers the highest seq per Message-ID.

import (
	"io"
	"mime/quotedprintable"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var wireEncodedRe = regexp.MustCompile(`=[0-9A-F]{2}`)

// looksWireEncoded is deliberately conservative: a body with a handful of =XX
// escapes or a soft line break is encoded; one stray "=" is not.
func looksWireEncoded(s string) bool {
	if strings.Contains(s, "=\r\n") || strings.Contains(s, "=\n") {
		return true
	}
	return len(wireEncodedRe.FindAllString(s, 4)) >= 3
}

func (c *crmStore) repairMessageEncoding(apply bool) (map[string]any, error) {
	base := filepath.Join(c.clientDir, "crm", "messages")
	months, _ := os.ReadDir(base)
	repaired := []any{}
	scanned := 0
	for _, m := range months {
		rows := readJSONLines(filepath.Join(base, m.Name(), "messages.jsonl"))
		for _, r := range rows {
			scanned++
			body := mStr(r, "body_text")
			if body == "" || !looksWireEncoded(body) || mStr(r, "provenance") == "repair_qp_decode" {
				continue
			}
			dec, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(body)))
			if err != nil {
				continue
			}
			if string(dec) == body {
				continue
			}
			entry := map[string]any{"rfc_message_id": mStr(r, "rfc_message_id"),
				"seq": mInt(r, "seq", 0), "chars_before": len(body), "chars_after": len(string(dec))}
			if apply {
				fix := map[string]any{}
				for k, v := range r {
					fix[k] = v
				}
				delete(fix, "seq")
				delete(fix, "id")
				delete(fix, "ts")
				fix["body_text"] = string(dec)
				fix["supersedes"] = mInt(r, "seq", 0)
				fix["provenance"] = "repair_qp_decode"
				if _, err := c.a.appendLog("messages", fix); err != nil {
					return nil, err
				}
			}
			repaired = append(repaired, entry)
		}
	}
	return map[string]any{"scanned": scanned, "wire_encoded": len(repaired),
		"applied": apply, "rows": repaired}, nil
}
