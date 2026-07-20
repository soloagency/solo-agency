package main

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"
)

// reply_poller.go — T1a: in the PERSISTENT bridge, detect campaign replies within
// minutes (not once a day) and notify the operator immediately. It reuses the
// exact same sync + classifier the daily run uses (gmailCmdSync), so a reply is
// found, the sequence frozen, and the CRM activity written identically — this
// poller only adds "run it on a timer" + "notify on genuinely new replies".
//
// Deliberately does NOT touch gmailCmdSync (that path is live-validated). The
// only new state is a small per-client dedup set so a reply is notified once,
// even if the daily run and this poller both sync the same sendbox (their
// imap_uid_cursor writes are unlocked and could reprocess one UID — the dedup
// set is the backstop, keyed by lead+activity_seq which is unique per client).

const (
	replyPollInterval = 5 * time.Minute
	replyPollMaxMsgs  = 300  // per sendbox per tick; backlog carries to the next tick
	replyNotifyKeep   = 1000 // bounded dedup memory per client
)

var replyPollRunning sync.Mutex // skip a tick if the previous one is still working

// startReplyPoller launches the background ticker (persistent mode only).
func (b *bridge) startReplyPoller(stop <-chan struct{}) {
	go func() {
		// small initial delay so first-boot baseline syncs settle before we poll
		select {
		case <-stop:
			return
		case <-time.After(30 * time.Second):
		}
		ticker := time.NewTicker(replyPollInterval)
		defer ticker.Stop()
		b.pollCampaignReplies()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				b.pollCampaignReplies()
			}
		}
	}()
}

// pollCampaignReplies syncs every client's sendboxes and notifies on new replies.
func (b *bridge) pollCampaignReplies() {
	if !replyPollRunning.TryLock() {
		return // previous tick still running (slow IMAP); skip this one
	}
	defer replyPollRunning.Unlock()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("reply-poller: recovered panic: %v", r)
		}
	}()

	for _, c := range b.uiClients() {
		clientDir := filepath.Join(c.Path, "outreach")
		boxes := mapsOf(mList(loadSendboxesDoc(clientDir), "sendboxes"))
		if len(boxes) == 0 {
			continue
		}
		var fresh []map[string]any
		for _, sb := range boxes {
			slug := mStr(sb, "slug")
			if slug == "" || mStr(sb, "status") == "needs_reauth" {
				continue // a box needing re-auth can't sync; skip quietly
			}
			res, err := gmailCmdSync(clientDir, slug, replyPollMaxMsgs)
			if err != nil {
				log.Printf("reply-poller: sync %s/%s: %v", c.Slug, slug, err)
				continue
			}
			for _, rv := range mList(res, "replies_untriaged") {
				if rm, ok := rv.(map[string]any); ok {
					fresh = append(fresh, rm)
				}
			}
		}
		if len(fresh) == 0 {
			continue
		}
		unseen := b.filterUnnotifiedReplies(clientDir, fresh)
		if len(unseen) > 0 {
			b.notifyReplies(c, clientDir, unseen)
		}
	}
}

// filterUnnotifiedReplies drops replies already notified (survives restarts via a
// small JSON set) and records the rest as seen. Key = lead_id|activity_seq.
func (b *bridge) filterUnnotifiedReplies(clientDir string, replies []map[string]any) []map[string]any {
	path := filepath.Join(clientDir, ".reply_notify_seen.json")
	doc, _ := readJSONFile(path)
	if doc == nil {
		doc = map[string]any{}
	}
	seenList := mList(doc, "seen")
	seen := map[string]bool{}
	for _, k := range seenList {
		seen[fmt.Sprint(k)] = true
	}
	var out []map[string]any
	for _, r := range replies {
		key := mStr(r, "lead_id") + "|" + fmt.Sprint(r["activity_seq"])
		if seen[key] {
			continue
		}
		seen[key] = true
		seenList = append(seenList, key)
		out = append(out, r)
	}
	if len(out) == 0 {
		return nil
	}
	if len(seenList) > replyNotifyKeep {
		seenList = seenList[len(seenList)-replyNotifyKeep:]
	}
	doc["seen"] = seenList
	doc["updated_at"] = nowISO()
	if err := atomicWriteFile(path, marshalIndentJSON(doc)); err != nil {
		log.Printf("reply-poller: persist dedup %s: %v", clientDir, err)
	}
	return out
}

// notifyReplies composes ONE operator notification for a batch of new replies and
// sends it through the client's configured provider (WideCast). Degrades to
// local_path_only when no provider is configured — never fails the poll.
func (b *bridge) notifyReplies(c uiClient, clientDir string, replies []map[string]any) {
	store := newCrmStore(clientDir)
	lines := make([]string, 0, len(replies))
	for i, r := range replies {
		if i >= 8 {
			lines = append(lines, fmt.Sprintf("...and %d more", len(replies)-8))
			break
		}
		who := mStr(r, "from")
		if ct := store.getContact(mStr(r, "lead_id")); ct != nil {
			if n := contactName(ct); n != "" {
				who = n
			}
		}
		camp := mStr(r, "campaign")
		subj := mStr(r, "subject")
		line := "- " + who
		if camp != "" {
			line += " (" + camp + ")"
		}
		if subj != "" {
			line += ": " + subj
		}
		lines = append(lines, line)
	}
	n := len(replies)
	noun := "reply"
	if n > 1 {
		noun = "replies"
	}
	subject := fmt.Sprintf("%s: %d new %s", c.Slug, n, noun)
	message := fmt.Sprintf("%d new campaign %s just arrived for %s. A reply freezes that lead's sequence automatically; review and respond on the Approvals/CRM page.\n\n%s",
		n, noun, c.Slug, joinLines(lines))

	cfg := filepath.Join(c.Path, "integrations", "providers", "provider_config.local.json")
	defaults := filepath.Join(b.uiDataRoot, "provider_defaults.json")
	logPath := filepath.Join(b.uiDataRoot, "notifications", "notification_log.md")
	na := notifyArgs{
		providerArgs: providerArgs{Config: cfg, Defaults: defaults},
		Event:        "campaign_reply_detected",
		Subject:      subject,
		Message:      message,
		Log:          logPath,
		Agent:        "Reply Poller",
		ActionNeeded: "Respond to the new repl" + map[bool]string{true: "ies", false: "y"}[n > 1] + " on the bridge Approvals/CRM page.",
	}
	if _, err := providerCmdNotify(na); err != nil {
		log.Printf("reply-poller: notify %s: %v", c.Slug, err)
		return
	}
	log.Printf("reply-poller: %s notified of %d new %s", c.Slug, n, noun)
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}
