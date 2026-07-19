package main

// imap.go — minimal hand-rolled IMAP4rev1 client over implicit TLS, exactly
// the surface gmail_client.py uses (docs/UI_DESIGN.md §8 G3): LOGIN,
// SELECT INBOX, UID SEARCH, UID FETCH <uid> (RFC822), LOGOUT. No IDLE, no
// APPEND, no STORE. Stdlib-only (crypto/tls + bufio).

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type imapClient struct {
	conn   net.Conn
	r      *bufio.Reader
	tagSeq int
}

func imapDial(host string, port int, timeout time.Duration) (*imapClient, error) {
	d := &net.Dialer{Timeout: timeout}
	conn, err := tls.DialWithDialer(d, "tcp", fmt.Sprintf("%s:%d", host, port), &tls.Config{ServerName: host})
	if err != nil {
		return nil, err
	}
	c := &imapClient{conn: conn, r: bufio.NewReader(conn)}
	conn.SetDeadline(time.Now().Add(timeout))
	if _, err := c.r.ReadString('\n'); err != nil { // greeting
		conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *imapClient) nextTag() string {
	c.tagSeq++
	return fmt.Sprintf("a%03d", c.tagSeq)
}

// command sends one command and collects untagged lines (with any literals
// inlined) until the tagged completion; returns the untagged lines and the
// tagged status line.
func (c *imapClient) command(format string, args ...any) ([]string, string, error) {
	tag := c.nextTag()
	cmd := fmt.Sprintf(format, args...)
	c.conn.SetDeadline(time.Now().Add(60 * time.Second))
	if _, err := fmt.Fprintf(c.conn, "%s %s\r\n", tag, cmd); err != nil {
		return nil, "", err
	}
	var untagged []string
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			return nil, "", err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(trimmed, tag+" ") {
			status := strings.TrimPrefix(trimmed, tag+" ")
			if !strings.HasPrefix(status, "OK") {
				return untagged, status, fmt.Errorf("imap: %s -> %s", strings.Fields(cmd)[0], status)
			}
			return untagged, status, nil
		}
		// literal? {N} at end of line — read N raw bytes and keep them attached
		if m := imapLiteralRe.FindStringSubmatch(trimmed); m != nil {
			n, _ := strconv.Atoi(m[1])
			buf := make([]byte, n)
			if _, err := io.ReadFull(c.r, buf); err != nil {
				return nil, "", err
			}
			untagged = append(untagged, trimmed+"\x00"+string(buf))
			continue
		}
		untagged = append(untagged, trimmed)
	}
}

var imapLiteralRe = regexp.MustCompile(`\{(\d+)\}$`)

func imapQuote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func (c *imapClient) login(user, pass string) error {
	_, _, err := c.command("LOGIN %s %s", imapQuote(user), imapQuote(pass))
	return err
}

func (c *imapClient) selectInbox() error {
	_, _, err := c.command("SELECT INBOX")
	return err
}

// uidSearch returns the UID list from `UID SEARCH <criteria>`.
func (c *imapClient) uidSearch(criteria string) ([]int, error) {
	untagged, _, err := c.command("UID SEARCH %s", criteria)
	if err != nil {
		return nil, err
	}
	var uids []int
	for _, line := range untagged {
		if strings.HasPrefix(line, "* SEARCH") {
			for _, f := range strings.Fields(strings.TrimPrefix(line, "* SEARCH")) {
				if n, err := strconv.Atoi(f); err == nil {
					uids = append(uids, n)
				}
			}
		}
	}
	return uids, nil
}

// uidFetchRFC822 returns the raw RFC822 bytes for one UID (nil when the
// server returned no literal — caller treats that as a transient miss).
func (c *imapClient) uidFetchRFC822(uid int) ([]byte, error) {
	untagged, _, err := c.command("UID FETCH %d (RFC822)", uid)
	if err != nil {
		return nil, err
	}
	for _, line := range untagged {
		if i := strings.IndexByte(line, '\x00'); i >= 0 && strings.Contains(line[:i], "FETCH") {
			return []byte(line[i+1:]), nil
		}
	}
	return nil, nil
}

func (c *imapClient) logout() {
	_, _, _ = c.command("LOGOUT")
	c.conn.Close()
}
