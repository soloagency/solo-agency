package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func commentFixture(t *testing.T) *crmStore {
	t.Helper()
	root := t.TempDir()
	store := newCrmStore(filepath.Join(root, "clients", "acme", "biz_loc", "outreach"))
	if err := store.initTree(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.createCampaign("comments", map[string]any{
		"channel_strategy": "comment", "status": "active",
		"goal": map[string]any{"description": "answer questions usefully"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.campaignUpdate("comments", map[string]any{
		"audience.groups": []any{"https://www.facebook.com/groups/764877593708803"},
	}); err != nil {
		t.Fatal(err)
	}
	return store
}

func commentArgs(post string) channelDraftArgs {
	return channelDraftArgs{
		PostURL: post, GroupURL: "https://www.facebook.com/groups/764877593708803",
		PostAuthor: "Fictional Author", PostExcerpt: "a question about contracting",
		BodyText: "Here is the part most people miss…",
	}
}

// TestChannelDraftWriteHappyPath: a comment draft carries the POST as its target
// and NO contact — the operator ruled on 2026-08-17 that a commented author never
// becomes a CRM record, so nothing in this path may need one.
func TestChannelDraftWriteHappyPath(t *testing.T) {
	store := commentFixture(t)
	s := defaultSystemSettings()
	res, err := store.channelDraftWrite("comments",
		commentArgs("https://www.facebook.com/groups/764877593708803/posts/3185530214976850/"), s)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	d, err := readJSONFile(mStr(res, "path"))
	if err != nil {
		t.Fatal(err)
	}
	if mStr(d, "lead_id") != "" {
		t.Fatalf("a comment draft must not carry a lead_id, got %q", mStr(d, "lead_id"))
	}
	if mStr(d, "channel") != "comment" || mStr(d, "status") != "pending_approval" {
		t.Fatalf("channel/status: %v %v", d["channel"], d["status"])
	}
	if mStr(d, "to") != mStr(d, "post_url") {
		t.Fatal("`to` must address the post — the approvals screen reads it")
	}
	if !strings.HasPrefix(mStr(d, "id"), "draft_") || len(mStr(d, "id")) < 20 {
		t.Fatalf("id must be a real ULID, got %q", mStr(d, "id"))
	}
	// It has to be visible to the operator, or the whole step is pointless.
	if got := len(store.listPendingDrafts("comments")); got != 1 {
		t.Fatalf("pending drafts = %d, want 1", got)
	}
}

// TestChannelDraftWriteRefusals: every gate the hand-written route had none of.
func TestChannelDraftWriteRefusals(t *testing.T) {
	s := defaultSystemSettings()
	cases := []struct {
		name, want string
		mutate     func(*channelDraftArgs)
	}{
		{"group not chosen by the operator", "group_not_in_campaign", func(a *channelDraftArgs) {
			a.GroupURL = "https://www.facebook.com/groups/111111111111111"
		}},
		{"post belongs to another group", "post_outside_group", func(a *channelDraftArgs) {
			a.PostURL = "https://www.facebook.com/groups/222222222222222/posts/3185530214976850/"
		}},
		{"no post id to pin", "unpinnable_post", func(a *channelDraftArgs) {
			a.PostURL = "https://www.facebook.com/groups/764877593708803/"
		}},
		{"nothing to say", "empty_body", func(a *channelDraftArgs) { a.BodyText = "   " }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := commentFixture(t)
			args := commentArgs("https://www.facebook.com/groups/764877593708803/posts/3185530214976850/")
			tc.mutate(&args)
			_, err := store.channelDraftWrite("comments", args, s)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %s, got %v", tc.want, err)
			}
		})
	}
}

// TestChannelDraftWriteNeverTwiceOnOnePost: the draft leaves pending_approval once
// published, so the memory of "we already answered this post" cannot live in the
// outbox — it lives in commented_posts.json and must survive the draft.
func TestChannelDraftWriteNeverTwiceOnOnePost(t *testing.T) {
	store := commentFixture(t)
	s := defaultSystemSettings()
	post := "https://www.facebook.com/groups/764877593708803/posts/pfbid02xk9qz7m4rt6vh3n8pq/"
	if _, err := store.channelDraftWrite("comments", commentArgs(post), s); err != nil {
		t.Fatalf("first write: %v", err)
	}
	_, err := store.channelDraftWrite("comments", commentArgs(post), s)
	if err == nil || !strings.Contains(err.Error(), "already_drafted") {
		t.Fatalf("want already_drafted, got %v", err)
	}
	// A pfbid post must be pinned by the POST token, never by the group id — the
	// same confusion that let a deleted post pass the collector's drift guard.
	if id := canonicalPostID(post); id != "pfbid02xk9qz7m4rt6vh3n8pq" {
		t.Fatalf("canonical post id = %q", id)
	}
}

// TestChannelDraftCapacityBound: drafting follows what the campaign can actually
// PUBLISH, not daily_quota. leadup-comment lives at 1 group × 1 comment/group/day
// against a daily_quota of 40 — drafting to the quota would have asked for forty
// decisions to ship one comment.
func TestChannelDraftCapacityBound(t *testing.T) {
	store := commentFixture(t)
	s := defaultSystemSettings()
	cfg := store.getCampaign("comments")
	per, reason := commentCapacityPerDay(cfg, s)
	if per != 1 {
		t.Fatalf("one group × 1 comment/group/day = 1, got %d (%s)", per, reason)
	}
	ceiling := per * channelDraftHorizonDays
	for i := 0; i < ceiling; i++ {
		post := "https://www.facebook.com/groups/764877593708803/posts/318553021497685" + string(rune('0'+i)) + "/"
		if _, err := store.channelDraftWrite("comments", commentArgs(post), s); err != nil {
			t.Fatalf("draft %d within ceiling refused: %v", i+1, err)
		}
	}
	_, err := store.channelDraftWrite("comments",
		commentArgs("https://www.facebook.com/groups/764877593708803/posts/9999999999999999/"), s)
	if err == nil || !strings.Contains(err.Error(), "capacity_horizon_full") {
		t.Fatalf("want capacity_horizon_full past the ceiling, got %v", err)
	}
	// More groups is the honest way to raise it — not a bigger quota number.
	if _, uerr := store.campaignUpdate("comments", map[string]any{"audience.groups": []any{
		"https://www.facebook.com/groups/764877593708803",
		"https://www.facebook.com/groups/764877593708804",
	}}); uerr != nil {
		t.Fatal(uerr)
	}
	if per2, _ := commentCapacityPerDay(store.getCampaign("comments"), s); per2 != 2 {
		t.Fatalf("adding a group must raise capacity, got %d", per2)
	}
}

// TestChannelDraftWriteWrongChannel: this writer is for post-targeted channels
// only; an email campaign must keep going through draftWrite and its gates.
func TestChannelDraftWriteWrongChannel(t *testing.T) {
	store := commentFixture(t)
	if _, err := store.createCampaign("mail", map[string]any{"channel_strategy": "email_first"}); err != nil {
		t.Fatal(err)
	}
	_, err := store.channelDraftWrite("mail",
		commentArgs("https://www.facebook.com/groups/764877593708803/posts/3185530214976850/"),
		defaultSystemSettings())
	if err == nil || !strings.Contains(err.Error(), "channel_mismatch") {
		t.Fatalf("want channel_mismatch, got %v", err)
	}
}

func postFixture(t *testing.T) *crmStore {
	t.Helper()
	root := t.TempDir()
	store := newCrmStore(filepath.Join(root, "clients", "acme", "biz_loc", "outreach"))
	if err := store.initTree(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.createCampaign("posts", map[string]any{
		"channel_strategy": "post", "status": "active",
		"goal": map[string]any{"description": "be the name people recognise"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.campaignUpdate("posts", map[string]any{
		"audience.groups": []any{
			"https://www.facebook.com/groups/764877593708803",
			"https://www.facebook.com/groups/764877593708804",
		}}); err != nil {
		t.Fatal(err)
	}
	return store
}

// TestGroupPostDraft: a group post answers nothing, so it needs no permalink —
// and its ceiling is per ACCOUNT, so listing more groups must not buy more posts.
func TestGroupPostDraft(t *testing.T) {
	store := postFixture(t)
	s := defaultSystemSettings()
	cfg := store.getCampaign("posts")
	per, reason := channelCapacityPerDay("post", cfg, s)
	if per != s.PostsPerAccountPerDay {
		t.Fatalf("post capacity = %d (%s), want %d", per, reason, s.PostsPerAccountPerDay)
	}
	// The load-bearing property: adding a group buys more COMMENTS and must buy no
	// more POSTS. Exposure per post does not divide by how many groups are listed.
	if _, err := store.campaignUpdate("posts", map[string]any{"audience.groups": []any{
		"https://www.facebook.com/groups/764877593708803",
		"https://www.facebook.com/groups/764877593708804",
		"https://www.facebook.com/groups/764877593708805",
	}}); err != nil {
		t.Fatal(err)
	}
	cfg3 := store.getCampaign("posts")
	perComment2, _ := channelCapacityPerDay("comment", cfg3, s)
	perPost2, _ := channelCapacityPerDay("post", cfg3, s)
	if perComment2 != 3 {
		t.Fatalf("three groups must give 3 comments/day, got %d", perComment2)
	}
	if perPost2 != per {
		t.Fatalf("a third group must not raise post capacity: %d -> %d", per, perPost2)
	}
	args := channelDraftArgs{GroupURL: "https://www.facebook.com/groups/764877593708803",
		BodyText: "Three line items quotes usually differ on, and why.",
		PostExcerpt: "read the last week of the group"}
	res, err := store.channelDraftWrite("posts", args, s)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	d, _ := readJSONFile(mStr(res, "path"))
	if mStr(d, "channel") != "post" || mStr(d, "post_url") != "" {
		t.Fatalf("a group post has no post_url: %v / %v", d["channel"], d["post_url"])
	}
	if mStr(d, "to") != args.GroupURL {
		t.Fatalf("`to` must address the group, got %v", d["to"])
	}
	// The same text must never go out twice, even into a different group.
	args2 := args
	args2.GroupURL = "https://www.facebook.com/groups/764877593708804"
	if _, err := store.channelDraftWrite("posts", args2, s); err == nil ||
		!strings.Contains(err.Error(), "already_drafted") {
		t.Fatalf("want already_drafted for repeated text, got %v", err)
	}
}
