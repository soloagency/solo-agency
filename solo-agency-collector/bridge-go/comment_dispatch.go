package main

// comment_dispatch.go — "approval is the command" for the comment channel.
//
// Publishing a comment needs no language model: the draft already carries the
// permalink and the exact text, and the collector already has fb.post.comment.
// So the operator pressing Approve on /ui/{client}/approvals IS the instruction
// to publish, and the bridge carries it out itself — no run, no agent, no wait.
// (Operator ruling 2026-08-17. The bridge does NOT decide WHETHER or WHAT to
// publish; it only executes a decision a human just made, which is why this is
// not the publishing daemon the operator rejected.)
//
// The one thing that must not be naive: approving five drafts in a row would
// otherwise fire five comments in ten seconds, which is what a bot looks like.
// So an approval enters a queue with a due time — the first is due now, the rest
// are spaced by the operator's publish gap — and the per-group daily cap is
// finally read by something. A capped item is not lost; it comes due after the
// local-day rollover.
//
// Freshness of the target post is deliberately NOT checked yet (operator ruling
// 2026-08-17: "trước mắt đừng quan tâm bài quá cũ, cứ duyệt là comment ngay").

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// A claimed job that never reports back. Shorter than harvest's 20m: a comment
	// is one page load and one click.
	commentJobStaleAfter = 12 * time.Minute
	commentJobTTLMinutes = 30
	commentDispatchTick  = 20 * time.Second
)

type commentQueued struct {
	DraftID    string `json:"draft_id"`
	Channel    string `json:"channel"` // comment | post
	GroupURL   string `json:"group_url"`
	Campaign   string `json:"campaign"`
	PostURL    string `json:"post_url"`
	GroupID    string `json:"group_id"`
	Body       string `json:"body_text"`
	ApprovedAt string `json:"approved_at"`
	DueAt      string `json:"due_at"`
	RunID      string `json:"run_id,omitempty"`
	Box        string `json:"box,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
}

type commentDispatchState struct {
	SchemaVersion int                      `json:"schema_version"`
	Day           string                   `json:"day"` // local day the counters belong to
	LastPublishAt string                   `json:"last_publish_at,omitempty"`
	PerGroup      map[string]int           `json:"per_group_today"`
	Queue         []commentQueued          `json:"queue"`
	InFlight      map[string]commentQueued `json:"in_flight"`
	Totals        map[string]int           `json:"totals"`
}

// publishGap: the operator's spacing floor (/ui/settings → "Minutes between two
// published actions"), default 5. Read fresh on every approval so a change takes
// effect on the next click rather than at the next restart.
func publishGap(s systemSettings) time.Duration {
	n := s.PublishGapMinutes
	if n <= 0 {
		n = 5
	}
	return time.Duration(n) * time.Minute
}

func commentDispatchPath(outreachDir string) string {
	return filepath.Join(outreachDir, "comment_dispatch.json")
}

// withCommentDispatch: the client's publish queue, under flock. The day rollover
// lives INSIDE the load closure so no caller can forget it (the harvest ledger's
// pattern) — otherwise yesterday's per-group counts would bench a group forever.
func withCommentDispatch(outreachDir string, now time.Time, fn func(*commentDispatchState) error) (*commentDispatchState, error) {
	st := &commentDispatchState{}
	err := withLockedJSON(commentDispatchPath(outreachDir),
		func(raw []byte) error {
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, st); err != nil {
					return fmt.Errorf("comment_dispatch.json is corrupt: %w", err)
				}
			}
			if st.SchemaVersion == 0 {
				st.SchemaVersion = 1
			}
			if st.PerGroup == nil {
				st.PerGroup = map[string]int{}
			}
			if st.InFlight == nil {
				st.InFlight = map[string]commentQueued{}
			}
			if st.Totals == nil {
				st.Totals = map[string]int{}
			}
			if day := now.Format("2006-01-02"); st.Day != day {
				st.Day = day
				st.PerGroup = map[string]int{}
			}
			return nil
		},
		func() ([]byte, error) { return json.MarshalIndent(st, "", "  ") },
		func() error { return fn(st) },
	)
	if err != nil {
		return nil, err
	}
	return st, nil
}

// findPendingDraft locates a draft awaiting approval anywhere under the client's
// campaigns. Approval arrives with an id and nothing else.
func findPendingDraft(outreachDir, draftID string) (path string, doc map[string]any, campaign string) {
	base := filepath.Join(outreachDir, "campaigns")
	camps, err := os.ReadDir(base)
	if err != nil {
		return "", nil, ""
	}
	want := draftID + ".json"
	for _, camp := range camps {
		if !camp.IsDir() {
			continue
		}
		root := filepath.Join(base, camp.Name(), "outbox", "pending_approval")
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, werr error) error {
			if werr != nil || d.IsDir() || d.Name() != want {
				return nil
			}
			if rec, rerr := readJSONFile(p); rerr == nil {
				path, doc, campaign = p, rec, camp.Name()
			}
			return filepath.SkipAll
		})
		if doc != nil {
			return path, doc, campaign
		}
	}
	return "", nil, ""
}

// approveCommentDraft applies an operator approval to a COMMENT draft in-process
// and queues it for publication. Returns handled=false for anything else (email
// and DM keep going through ui_inbox + the run's ingest, untouched).
func (b *bridge) approveCommentDraft(outreachDir, draftID string, now time.Time) (res map[string]any, handled bool, err error) {
	path, d, campaign := findPendingDraft(outreachDir, draftID)
	channel := mStr(d, "channel")
	if d == nil || (channel != "comment" && channel != "post") {
		return nil, false, nil
	}
	postURL := mStr(d, "post_url")
	groupURL := mStr(d, "group_url")
	body := strings.TrimSpace(mStr(d, "body_text"))
	if body == "" {
		return nil, true, storageErrf("draft %s has no body_text — nothing to publish", draftID)
	}
	if channel == "comment" && postURL == "" {
		return nil, true, storageErrf("draft %s has no post_url — a comment needs the post it answers", draftID)
	}
	if channel == "post" && groupURL == "" {
		return nil, true, storageErrf("draft %s has no group_url — a group post needs the group it goes into", draftID)
	}
	groupID := groupIDFromURL(groupURL)

	// Move it to the ready queue exactly as ingestUIDecisions would, so both paths
	// leave identical state on disk.
	store := newCrmStore(outreachDir)
	cd, cerr := store.campaignDir(campaign)
	if cerr != nil {
		return nil, true, cerr
	}
	d["status"] = "approved"
	d["decided_at"] = nowISO()
	d["decided_by"] = "ui"
	approvedDir := filepath.Join(cd, "outbox", "approved")
	if mkerr := os.MkdirAll(approvedDir, 0o755); mkerr != nil {
		return nil, true, mkerr
	}
	if werr := atomicWriteFile(path, marshalIndentJSON(d)); werr != nil {
		return nil, true, werr
	}
	if rerr := os.Rename(path, filepath.Join(approvedDir, draftID+".json")); rerr != nil {
		return nil, true, rerr
	}
	_ = store.approvalLog(d, "approve", "ui", "")

	settings := loadSystemSettings(pipelineRootFromClientDir(outreachDir))
	var due time.Time
	var note string
	if _, serr := withCommentDispatch(outreachDir, now, func(st *commentDispatchState) error {
		for _, q := range st.Queue {
			if q.DraftID == draftID {
				return storageErrf("draft %s is already queued to publish", draftID)
			}
		}
		due = now
		// Spacing first: never two comments closer than the operator's gap.
		if last, perr := time.Parse(time.RFC3339, st.LastPublishAt); perr == nil {
			if t := last.Add(publishGap(settings)); t.After(due) {
				due, note = t, "spaced behind the previous comment"
			}
		}
		for _, q := range st.Queue {
			if t, perr := time.Parse(time.RFC3339, q.DueAt); perr == nil {
				if s := t.Add(publishGap(settings)); s.After(due) {
					due, note = s, "spaced behind the comments already queued"
				}
			}
		}
		// Then the caps the operator set and nothing has ever read.
		capped := ""
		if channel == "post" {
			// Posting is capped per ACCOUNT, not per group: one standalone piece of
			// content in front of a whole group is the same exposure whichever group
			// it goes into, so listing more groups must not buy more posts.
			perAccount := settings.PostsPerAccountPerDay
			if perAccount <= 0 {
				perAccount = 2
			}
			if st.PerGroup["__posts__"] >= perAccount {
				capped = fmt.Sprintf("this account's daily limit of %d new group post(s) is already used", perAccount)
			}
		} else {
			perGroup := settings.CommentsPerGroupPerDay
			if perGroup <= 0 {
				perGroup = 1
			}
			if groupID != "" && st.PerGroup[groupID] >= perGroup {
				capped = fmt.Sprintf("that group's daily limit (%d) is already used", perGroup)
			} else if groupID != "" && st.PerGroup[groupID] == 0 &&
				settings.CommentGroupsPerAccountPerDay > 0 && len(st.PerGroup) >= settings.CommentGroupsPerAccountPerDay {
				capped = fmt.Sprintf("this account has already commented in %d groups today", settings.CommentGroupsPerAccountPerDay)
			}
		}
		if capped != "" {
			midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
			if midnight.After(due) {
				due = midnight
			}
			note = capped + " — it publishes after midnight"
		}
		st.Queue = append(st.Queue, commentQueued{DraftID: draftID, Channel: channel, Campaign: campaign,
			PostURL: postURL, GroupURL: groupURL, GroupID: groupID, Body: body,
			Box: mStr(d, "collector"), ApprovedAt: nowISO(), DueAt: due.Format(time.RFC3339)})
		st.Totals["queued"]++
		return nil
	}); serr != nil {
		return nil, true, serr
	}
	out := map[string]any{"ok": true, "draft_id": draftID, "campaign": campaign,
		"publishes": "now", "due_at": due.Format(time.RFC3339)}
	if !due.After(now.Add(2 * time.Second)) {
		out["note"] = "publishing now"
	} else {
		out["publishes"] = due.Local().Format("15:04")
		out["note"] = strOr(note, "queued")
	}
	return out, true, nil
}

// ---- the drain -------------------------------------------------------------

func (b *bridge) startCommentDispatch(stop <-chan struct{}) {
	go func() {
		select {
		case <-stop:
			return
		case <-time.After(30 * time.Second):
		}
		ticker := time.NewTicker(commentDispatchTick)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				b.commentDispatchStep(time.Now())
			}
		}
	}()
}

// commentReadJob: has the comment job finished, and did it actually post? A
// write counts as done only when the record says so — `status: done` AND
// `verified: true`; anything else is a failure to report, never a retry.
// commentReadJob reports whether the write left our hands. The collector
// distinguishes published from pending_admin_approval, and this layer
// deliberately does NOT: whether a group's admin holds, approves or rejects a
// post is outside the operator's control, so making it a state the system
// tracks would add a queue he can neither see nor act on (operator ruling
// 2026-08-17). We posted; that is the end of our part. The collector's own word
// is still recorded on the draft for traceability.
func (b *bridge) commentReadJob(runID string) (done bool, outcome, detail string) {
	matches, _ := filepath.Glob(filepath.Join(b.outputRoot, "*", "*", safeFilename(runID)))
	if len(matches) == 0 {
		return false, "", ""
	}
	dir := matches[0]
	status, err := readJSONFile(filepath.Join(dir, "collector_status.json"))
	if err != nil {
		return false, "", ""
	}
	st := mStr(status, "status")
	doneFlag, _ := status["completed"].(bool)
	if !doneFlag && st != "completed" && st != "failed" && st != "error" {
		return false, "", ""
	}
	for _, row := range readJSONLines(filepath.Join(dir, "private_data_points.jsonl")) {
		recs := mMap(row, "records")
		cap := mStr(recs, "capability")
		if cap != "fb.post.comment" && cap != "fb.group.post" {
			continue
		}
		for _, it := range mapsOf(mList(recs, "items")) {
			switch mStr(it, "status") {
			case "done":
				if truthy(it["verified"]) {
					return true, "published", strOr(mStr(it, "comment_url"), mStr(it, "post_url"))
				}
				return true, "failed", "the collector reported done but did not verify it"
			case "pending_admin_approval":
				// Out of our hands from here — counted as done, not as a pending queue.
				return true, "published", "submitted; this group holds new posts for an admin"
			default:
				return true, "failed", strOr(mStr(it, "error"), mStr(it, "status"))
			}
		}
	}
	return true, "failed", "the collector reported no write record"
}

func (b *bridge) commentDispatchStep(now time.Time) {
	for _, c := range b.uiClients() {
		outreachDir := filepath.Join(c.Path, "outreach")
		if _, err := os.Stat(commentDispatchPath(outreachDir)); err != nil {
			continue // this client has never approved a comment
		}
		b.commentDispatchClient(c, outreachDir, now)
	}
}

func (b *bridge) commentDispatchClient(c uiClient, outreachDir string, now time.Time) {
	st, err := withCommentDispatch(outreachDir, now, func(*commentDispatchState) error { return nil })
	if err != nil || st == nil || (len(st.Queue) == 0 && len(st.InFlight) == 0) {
		return
	}
	// 1. Resolve what is in flight. Reads happen outside the lock; the mutation
	//    that follows is the only thing holding it.
	type settledJob struct {
		item                   commentQueued
		outcome, detail, runID string
	}
	var settled []settledJob
	for runID, item := range st.InFlight {
		done, outcome, detail := b.commentReadJob(runID)
		if !done {
			if t, perr := time.Parse(time.RFC3339, item.StartedAt); perr == nil && now.Sub(t) > commentJobStaleAfter {
				settled = append(settled, settledJob{item, "failed", "the collector never reported back", runID})
			}
			continue
		}
		settled = append(settled, settledJob{item, outcome, detail, runID})
	}
	for _, o := range settled {
		b.commentRecordOutcome(outreachDir, o.item, o.outcome, o.detail)
		_, _ = withCommentDispatch(outreachDir, now, func(st *commentDispatchState) error {
			delete(st.InFlight, o.runID)
			st.Totals[o.outcome]++
			return nil
		})
	}

	// 2. One comment at a time per client, and only when due.
	st, err = withCommentDispatch(outreachDir, now, func(*commentDispatchState) error { return nil })
	if err != nil || st == nil || len(st.InFlight) > 0 || len(st.Queue) == 0 {
		return
	}
	head := st.Queue[0]
	if t, perr := time.Parse(time.RFC3339, head.DueAt); perr == nil && t.After(now) {
		return
	}
	// The publishing account is NOT interchangeable. Only the collector that READ the
	// group is proven to be a member of it, and a non-member is served the post with
	// no comment composer at all — which is exactly the failure measured live on
	// 2026-08-17. So publish from the box that did the reading, or from none: a
	// different account is a guess whose cost is paid in account standing.
	var box harvestCollector
	for _, live := range b.liveCollectors(now) {
		if live.ClientSlug != c.Slug {
			continue
		}
		if head.Box == "" || live.InstanceID == head.Box {
			box = live
			break
		}
	}
	if box.InstanceID == "" {
		// Nothing is lost and nothing is guessed: the item waits for its own account
		// to check in. Surfaced on the campaign page as the reason it has not gone out.
		_, _ = withCommentDispatch(outreachDir, now, func(st *commentDispatchState) error {
			if len(st.Queue) > 0 && st.Queue[0].DraftID == head.DraftID {
				if head.Box != "" {
					st.Totals["waiting_for_its_collector"] = 1
				} else {
					st.Totals["waiting_for_a_collector"] = 1
				}
			}
			return nil
		})
		return
	}
	capability, targetURL := "fb.post.comment", head.PostURL
	inputs := map[string]any{"post_url": head.PostURL, "text": head.Body}
	if head.Channel == "post" {
		capability, targetURL = "fb.group.post", head.GroupURL
		inputs = map[string]any{"group_url": head.GroupURL, "text": head.Body}
	}
	runID := fmt.Sprintf("publish_%s_%s", now.Format("20060102_150405"), safeFilename(head.DraftID))
	job := map[string]any{
		"run_id": runID, "job_type": "run_now", "client_slug": box.ClientSlug,
		"allowed_extension_instance_ids": []any{box.InstanceID},
		"extension_instance_id":          box.InstanceID,
		"run_now_ttl_minutes":            commentJobTTLMinutes,
		"sources": []any{map[string]any{
			"name": head.Channel + " " + head.Campaign, "url": targetURL, "platform": "facebook",
			"source_type": "public", "capability": capability,
			"inputs": inputs, "priority": "high",
		}},
		"pacing": map[string]any{"min_delay_seconds": 3, "max_delay_seconds": 6, "max_sources": 1, "scroll_steps": 0},
	}
	if _, eerr := b.enqueueRunNowPayload(job, now, "comment_dispatch"); eerr != nil {
		log.Printf("comment_dispatch[%s]: enqueue %s: %v", c.Slug, head.DraftID, eerr)
		return
	}
	_, _ = withCommentDispatch(outreachDir, now, func(st *commentDispatchState) error {
		if len(st.Queue) == 0 || st.Queue[0].DraftID != head.DraftID {
			return nil
		}
		it := st.Queue[0]
		it.RunID, it.Box, it.StartedAt = runID, box.InstanceID, now.Format(time.RFC3339)
		st.Queue = st.Queue[1:]
		st.InFlight[runID] = it
		st.LastPublishAt = now.Format(time.RFC3339)
		if it.Channel == "post" {
			st.PerGroup["__posts__"]++
		} else if it.GroupID != "" {
			st.PerGroup[it.GroupID]++
		}
		return nil
	})
}

// commentRecordOutcome closes the loop on disk: the draft stops being "approved
// and waiting" (which is what the capacity ceiling counts), and the post is
// marked so no future scan can answer it twice.
func (b *bridge) commentRecordOutcome(outreachDir string, item commentQueued, outcome, detail string) {
	store := newCrmStore(outreachDir)
	cd, err := store.campaignDir(item.Campaign)
	if err != nil {
		return
	}
	p := filepath.Join(cd, "outbox", "approved", item.DraftID+".json")
	movedBack := false
	if d, rerr := readJSONFile(p); rerr == nil {
		switch outcome {
		case "published":
			// Into the client's distribution log, beside the emails. The screen that
			// used to be "Sent" answers "what has this client put out", and a comment
			// or a group post belongs in that answer as much as an email does.
			_ = gmailAppendSentLog(outreachDir, item.Campaign, map[string]any{
				"channel": item.Channel, "campaign": item.Campaign, "draft_id": item.DraftID,
				"target_url": strOr(item.PostURL, item.GroupURL), "group_url": item.GroupURL,
				"collector": item.Box, "published_url": detail, "sent_at": nowISO(), "step": 1,
			})
			// "sent" is the status gmail.go uses for a published draft, and the
			// capacity count keys off it — a comment must age out of the queue the
			// same way an email does.
			d["status"] = "sent"
			d["posted_at"] = nowISO()
			if detail != "" {
				d["published_url"] = detail
			}
		default:
			// Back to the approval queue, not into a dead "blocked" pile. The failures
			// seen here are environmental — the account is not a member of the group,
			// the page never rendered — and every one of them is something the operator
			// fixes and then retries. Leaving it in approved/ with status blocked gave
			// him a draft he could see nowhere and re-approve never. The post is
			// released from the dedupe index in the same pass, so this stays consistent.
			d["status"] = "pending_approval"
			d["decided_at"] = ""
			d["decided_by"] = ""
			d["blocker"] = detail
			d["blocked_at"] = nowISO()
			back := filepath.Join(cd, "outbox", "pending_approval", todayStr(nowISO()))
			if os.MkdirAll(back, 0o755) == nil && atomicWriteFile(p, marshalIndentJSON(d)) == nil {
				_ = os.Rename(p, filepath.Join(back, item.DraftID+".json"))
			}
			// Deliberately NOT returning: the dedupe release below must still run, or a
			// draft handed back for another try would be refused as already_drafted.
			movedBack = true
		}
		d["updated_at"] = nowISO()
		if !movedBack {
			_ = atomicWriteFile(p, marshalIndentJSON(d))
		}
	}
	key := "body:" + bodyFingerprint(item.Body)
	if item.Channel != "post" {
		key = canonicalPostID(item.PostURL)
	}
	if key != "" {
		_, _ = withSeenPosts(cd, func(idx *seenPostIndex) error {
			e, ok := idx.Posts[key]
			if !ok {
				return nil
			}
			if outcome == "failed" {
				// Released, so a later pass may try again: an action that never landed
				// must not permanently burn its target.
				delete(idx.Posts, key)
				return nil
			}
			e.Status = "posted"
			idx.Posts[key] = e
			return nil
		})
	}
	log.Printf("publish_dispatch[%s/%s]: %s %s %s", item.Campaign, strOr(item.Channel, "comment"), item.DraftID, outcome, detail)
}
