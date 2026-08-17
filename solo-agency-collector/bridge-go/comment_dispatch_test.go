package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// draftForApproval writes a comment draft through the sanctioned writer and hands
// back its id, so these tests exercise the same shape the UI will see.
func draftForApproval(t *testing.T, store *crmStore, post string) string {
	t.Helper()
	res, err := store.channelDraftWrite("comments", commentArgs(post), defaultSystemSettings())
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	return mStr(res, "draft_id")
}

// TestApproveCommentPublishesImmediately: approval IS the command. The draft must
// leave pending_approval and be queued to publish in the same request — not wait
// for an agent run, which is what made a comment land on a thread that had moved on.
func TestApproveCommentPublishesImmediately(t *testing.T) {
	store := commentFixture(t)
	id := draftForApproval(t, store, "https://www.facebook.com/groups/764877593708803/posts/3185530214976850/")
	b := &bridge{}
	now := time.Now()

	res, handled, err := b.approveCommentDraft(store.clientDir, id, now)
	if err != nil || !handled {
		t.Fatalf("approve: handled=%v err=%v", handled, err)
	}
	if mStr(res, "note") != "publishing now" {
		t.Fatalf("first approval must publish now, got %v", res)
	}
	// The draft moved to the ready queue, stamped as a human decision.
	cd, _ := store.campaignDir("comments")
	d, rerr := readJSONFile(filepath.Join(cd, "outbox", "approved", id+".json"))
	if rerr != nil {
		t.Fatalf("draft did not move to approved/: %v", rerr)
	}
	if mStr(d, "status") != "approved" || mStr(d, "decided_by") != "ui" {
		t.Fatalf("status/decided_by = %v / %v", d["status"], d["decided_by"])
	}
	if got := len(store.listPendingDrafts("comments")); got != 0 {
		t.Fatalf("still pending: %d", got)
	}
	st, _ := withCommentDispatch(store.clientDir, now, func(*commentDispatchState) error { return nil })
	if len(st.Queue) != 1 || st.Queue[0].PostURL == "" || st.Queue[0].Body == "" {
		t.Fatalf("publish queue: %+v", st.Queue)
	}
}

// TestApproveCommentBurstIsSpaced: the operator approving several in a row must not
// fire several comments in seconds — that is exactly what a bot looks like.
func TestApproveCommentBurstIsSpaced(t *testing.T) {
	store := commentFixture(t)
	// Two groups so the per-group daily cap is not what does the spacing here.
	if _, err := store.campaignUpdate("comments", map[string]any{"audience.groups": []any{
		"https://www.facebook.com/groups/764877593708803",
		"https://www.facebook.com/groups/764877593708804",
	}}); err != nil {
		t.Fatal(err)
	}
	b := &bridge{}
	now := time.Now()
	a := draftForApproval(t, store, "https://www.facebook.com/groups/764877593708803/posts/3185530214976851/")
	args := commentArgs("https://www.facebook.com/groups/764877593708804/posts/3185530214976852/")
	args.GroupURL = "https://www.facebook.com/groups/764877593708804"
	res2, err := store.channelDraftWrite("comments", args, defaultSystemSettings())
	if err != nil {
		t.Fatal(err)
	}
	second := mStr(res2, "draft_id")

	if _, _, err := b.approveCommentDraft(store.clientDir, a, now); err != nil {
		t.Fatal(err)
	}
	out, _, err := b.approveCommentDraft(store.clientDir, second, now)
	if err != nil {
		t.Fatal(err)
	}
	if mStr(out, "note") == "publishing now" {
		t.Fatal("two approvals in a row must not both publish immediately")
	}
	st, _ := withCommentDispatch(store.clientDir, now, func(*commentDispatchState) error { return nil })
	if len(st.Queue) != 2 {
		t.Fatalf("queue = %d, want 2", len(st.Queue))
	}
	d0, _ := time.Parse(time.RFC3339, st.Queue[0].DueAt)
	d1, _ := time.Parse(time.RFC3339, st.Queue[1].DueAt)
	want := publishGap(defaultSystemSettings())
	if want != 5*time.Minute {
		t.Fatalf("the shipped default gap is %s, expected 5m", want)
	}
	if gap := d1.Sub(d0); gap < want {
		t.Fatalf("gap between queued comments = %s, want >= %s", gap, want)
	}
}

// TestApproveCommentGroupCapDefersToTomorrow: the per-group daily cap in
// /ui/settings finally has a reader. A capped approval is not lost — it comes due
// after the local-day rollover.
func TestApproveCommentGroupCapDefersToTomorrow(t *testing.T) {
	store := commentFixture(t)
	b := &bridge{}
	now := time.Now()
	first := draftForApproval(t, store, "https://www.facebook.com/groups/764877593708803/posts/3185530214976853/")
	if _, _, err := b.approveCommentDraft(store.clientDir, first, now); err != nil {
		t.Fatal(err)
	}
	// Pretend the first one published, so the group's single daily slot is spent.
	if _, err := withCommentDispatch(store.clientDir, now, func(st *commentDispatchState) error {
		st.PerGroup["764877593708803"] = defaultSystemSettings().CommentsPerGroupPerDay
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	second := draftForApproval(t, store, "https://www.facebook.com/groups/764877593708803/posts/3185530214976854/")
	out, _, err := b.approveCommentDraft(store.clientDir, second, now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mStr(out, "note"), "after midnight") {
		t.Fatalf("a capped approval must say when it publishes, got %v", out)
	}
	st, _ := withCommentDispatch(store.clientDir, now, func(*commentDispatchState) error { return nil })
	due, _ := time.Parse(time.RFC3339, st.Queue[len(st.Queue)-1].DueAt)
	if !due.After(now.Add(1 * time.Hour)) {
		t.Fatalf("capped item due at %s, expected after the day rolls over", due)
	}
}

// TestApproveNonCommentIsNotHandled: email and DM must keep going through the run's
// ingest — this path may never touch them.
func TestApproveNonCommentIsNotHandled(t *testing.T) {
	store := commentFixture(t)
	b := &bridge{}
	_, handled, err := b.approveCommentDraft(store.clientDir, "draft_does_not_exist", time.Now())
	if handled || err != nil {
		t.Fatalf("unknown draft: handled=%v err=%v", handled, err)
	}
}

// TestCommentOutcomeReleasesAFailedPost: a comment that never landed must not burn
// its target — the next scan may try again. A published one is remembered forever.
func TestCommentOutcomeReleasesAFailedPost(t *testing.T) {
	store := commentFixture(t)
	b := &bridge{}
	post := "https://www.facebook.com/groups/764877593708803/posts/3185530214976855/"
	id := draftForApproval(t, store, post)
	if _, _, err := b.approveCommentDraft(store.clientDir, id, time.Now()); err != nil {
		t.Fatal(err)
	}
	cd, _ := store.campaignDir("comments")
	item := commentQueued{DraftID: id, Campaign: "comments", PostURL: post}

	b.commentRecordOutcome(store.clientDir, item, "failed", "action_blocked")
	idx, _ := withSeenPosts(cd, func(*seenPostIndex) error { return nil })
	if _, still := idx.Posts[canonicalPostID(post)]; still {
		t.Fatal("a failed comment must release the post for a later attempt")
	}
	d, _ := readJSONFile(filepath.Join(cd, "outbox", "approved", id+".json"))
	if mStr(d, "status") != "blocked" || mStr(d, "blocker") != "action_blocked" {
		t.Fatalf("failed draft: %v / %v", d["status"], d["blocker"])
	}

	// And the published case: remembered, and no longer counted as waiting work.
	id2 := draftForApproval(t, store, post)
	if _, _, err := b.approveCommentDraft(store.clientDir, id2, time.Now()); err != nil {
		t.Fatal(err)
	}
	b.commentRecordOutcome(store.clientDir, commentQueued{DraftID: id2, Campaign: "comments", PostURL: post},
		"published", "https://www.facebook.com/groups/764877593708803/posts/3185530214976855/?comment_id=9")
	idx, _ = withSeenPosts(cd, func(*seenPostIndex) error { return nil })
	if idx.Posts[canonicalPostID(post)].Status != "posted" {
		t.Fatal("a published comment must be remembered as posted")
	}
	alive, _ := store.aliveChannelDrafts("comments")
	if alive != 0 {
		t.Fatalf("a published comment must stop counting against capacity, alive=%d", alive)
	}
}
