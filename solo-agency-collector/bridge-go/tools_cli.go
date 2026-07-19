package main

// tools_cli.go — G1 tool subcommands per docs/UI_DESIGN.md §8.
//
// CLI-compat contract: `<binary> tool <name> <subcommand> [flags]` mirrors the
// Python tool's flags and JSON stdout exactly, so playbooks can switch to
// binary-first with a python3 fallback and cross-validation can diff outputs.
//
// G1 ships verify-email (replaces outreach/tools/email_verify.py). Unlike the
// Python original, MX resolution uses net.LookupMX directly — no dig/nslookup
// shell-outs — with the same A/AAAA implicit-MX fallback per RFC 5321.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
)

var emailSyntaxRe = regexp.MustCompile(`^[A-Za-z0-9!#$%&'*+/=?^_` + "`" + `{|}~.-]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+$`)

func emailSyntaxOK(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" || strings.Count(email, "@") != 1 || strings.Contains(email, "..") {
		return false
	}
	if len(email) > 254 {
		return false
	}
	return emailSyntaxRe.MatchString(email)
}

// emailMXLookup mirrors email_verify.py mx_lookup: MX hosts, else implicit MX
// (domain resolves at all), else empty. Injectable resolvers for tests.
var lookupMX = net.LookupMX
var lookupHost = net.LookupHost

func emailMXLookup(domain string) []string {
	if mxs, err := lookupMX(domain); err == nil && len(mxs) > 0 {
		hosts := make([]string, 0, len(mxs))
		for _, mx := range mxs {
			h := strings.TrimSuffix(mx.Host, ".")
			if h != "" {
				hosts = append(hosts, h)
			}
		}
		if len(hosts) > 0 {
			return hosts
		}
	}
	if addrs, err := lookupHost(domain); err == nil && len(addrs) > 0 {
		return []string{domain} // implicit MX per RFC 5321
	}
	return []string{}
}

// emailCheckResult matches email_verify.py's JSON field set and order.
type emailCheckResult struct {
	Email    string   `json:"email"`
	SyntaxOK bool     `json:"syntax_ok"`
	MXOK     bool     `json:"mx_ok"`
	MXHosts  []string `json:"mx_hosts"`
	Status   string   `json:"status"`
	Note     string   `json:"note"`
}

func emailCheck(email string) emailCheckResult {
	email = strings.TrimSpace(email)
	res := emailCheckResult{Email: email, MXHosts: []string{}, Status: "unverified"}
	res.SyntaxOK = emailSyntaxOK(email)
	if !res.SyntaxOK {
		res.Status = "syntax_invalid"
		res.Note = "address failed syntax check"
		return res
	}
	at := strings.LastIndex(email, "@")
	domain := strings.ToLower(email[at+1:])
	hosts := emailMXLookup(domain)
	res.MXHosts = hosts
	res.MXOK = len(hosts) > 0
	if res.MXOK {
		res.Status = "mx_ok"
		res.Note = "domain can receive mail (mailbox existence NOT proven; guessed addresses still need API verification)"
	} else {
		res.Status = "mx_fail"
		res.Note = "no MX and domain does not resolve — will hard-bounce"
	}
	return res
}

func printJSONLine(v any) {
	out, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(string(out))
}

// maybeRunToolCLI handles `<binary> tool ...` and returns true when it ran
// (the caller must exit without starting the bridge server). Exit codes and
// flag names mirror the Python originals.
func maybeRunToolCLI(args []string) (handled bool, exitCode int) {
	if len(args) == 0 || args[0] != "tool" {
		return false, 0
	}
	usage := func() int {
		fmt.Fprintln(os.Stderr, "usage: tool verify-email check --email a@b.com | check-file --in emails.txt")
		fmt.Fprintln(os.Stderr, "       tool crm-store [--client-dir DIR | --pipeline R --client S] <subcommand> ...")
		return 2
	}
	if len(args) < 2 {
		return true, usage()
	}
	switch args[1] {
	case "crm-store":
		return true, runCrmStoreCLI(args[2:])
	case "import-leads":
		return true, runImportLeadsCLI(args[2:])
	case "render-report":
		return true, runRenderReportCLI(args[2:])
	case "provider":
		return true, runProviderCLI(args[2:])
	case "gmail":
		return true, runGmailCLI(args[2:])
	case "verify-email":
		if len(args) < 3 {
			return true, usage()
		}
		switch args[2] {
		case "check":
			email := flagValue(args[3:], "--email")
			if email == "" {
				return true, usage()
			}
			printJSONLine(emailCheck(email))
			return true, 0
		case "check-file":
			in := flagValue(args[3:], "--in")
			if in == "" {
				return true, usage()
			}
			f, err := os.Open(in)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return true, 1
			}
			defer f.Close()
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				addr := strings.TrimSpace(sc.Text())
				if addr != "" {
					printJSONLine(emailCheck(addr))
				}
			}
			return true, 0
		}
		return true, usage()
	}
	return true, usage()
}

func flagValue(args []string, name string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == name && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(args[i], name+"=") {
			return strings.TrimPrefix(args[i], name+"=")
		}
	}
	return ""
}
