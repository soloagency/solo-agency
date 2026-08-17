package main

// crm_channel_draft.go — the compose path for POST-TARGETED channels.
//
// Email drafting starts from a PERSON: pick a contact out of the pipeline,
// enrich them, write to their address (draftWrite). A comment runs the other
// way round — it starts from a POST inside a group the operator named, and the
// thing being addressed is that post, not anybody's inbox. draftWrite cannot
// express that: it resolves a contact, requires a usable email address, and
// refuses without one (crm_reports.go, "contact has no usable email to draft
// to"). Every author worth answering in a group is exactly the person who has
// no email on file, so the sanctioned writer had no path for this channel at
// all — which is why three comment drafts on 2026-08-15 were written into
// pending_approval/ by hand, outside every gate, with synthetic ids.
//
// Operator ruling 2026-08-17: a commented post's author NEVER becomes a CRM
// contact. Commenting is a visibility play, not a lead capture, so these drafts
// carry no lead_id and the CRM stays free of email-less shells. Deduplication
// therefore cannot lean on the contact registry and gets its own index, keyed
// by the post itself.
//
// Second ruling the same day: the drafting ceiling follows PUBLISH CAPACITY,
// not daily_quota. leadup-comment has one group and comments_per_group_per_day
// is 1, so it can publish one comment a day while its "Daily draft budget"
// field said 40 — drafting to 40 would have asked the operator for forty
// decisions to ship one comment, and thirty-nine drafts would have aged out
// unpublished. Capacity is computed from the same numbers the publish side is
// bounded by, so approving everything on the page is always a correct move.

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// channelDraftHorizonDays: how far ahead of publish capacity drafting may run.
// Two days of stock — enough that the operator always has something to approve
// after a day away, short enough that a draft is never published against a post
// that has gone cold.
const channelDraftHorizonDays = 2

type channelDraftArgs struct {
	PostURL     string
	GroupURL    string
	PostAuthor  string
	PostExcerpt string
	BodyText    string
	PostSeenAt  string // when the scan observed the post (RFC3339); freshness is judged from it
}

// ---- post identity ---------------------------------------------------------

var (
	fbGroupIDRe = regexp.MustCompile(`(?i)facebook\.com/groups/([0-9a-z._-]+)`)
	fbPostIDRe  = regexp.MustCompile(`(?i)/(?:posts|permalink)/(pfbid[0-9a-z]+|\d{6,})`)
	fbStoryRe   = regexp.MustCompile(`(?i)[?&]story_fbid=(pfbid[0-9a-z]+|\d{6,})`)
)

func groupIDFromURL(u string) string {
	if m := fbGroupIDRe.FindStringSubmatch(u); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// canonicalPostID pins the POST, never the group. Taking "the first long number
// in the url" would return the GROUP id for /groups/<gid>/posts/<pid> — the same
// mistake that let a deleted post pass the collector's drift guard, because the
// group id of course survives on the group's feed.
func canonicalPostID(u string) string {
	if m := fbPostIDRe.FindStringSubmatch(u); m != nil {
		return strings.ToLower(m[1])
	}
	if m := fbStoryRe.FindStringSubmatch(u); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// ---- the seen-post index ---------------------------------------------------

type seenPost struct {
	PostURL  string `json:"post_url"`
	GroupURL string `json:"group_url"`
	Author   string `json:"author,omitempty"`
	DraftID  string `json:"draft_id"`
	Status   string `json:"status"` // drafted | posted | dropped
	FirstAt  string `json:"first_at"`
}

type seenPostIndex struct {
	SchemaVersion int                 `json:"schema_version"`
	Posts         map[string]seenPost `json:"posts"`
}

func seenPostsPath(campaignDir string) string {
	return filepath.Join(campaignDir, "commented_posts.json")
}

// withSeenPosts: the campaign's memory of every post it has already written a
// comment for. It must outlive the draft — once published, the draft leaves
// pending_approval, and without this index the next scan would happily write a
// second comment onto the same post under the same account.
func withSeenPosts(campaignDir string, fn func(*seenPostIndex) error) (*seenPostIndex, error) {
	idx := &seenPostIndex{}
	err := withLockedJSON(seenPostsPath(campaignDir),
		func(raw []byte) error {
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, idx); err != nil {
					return fmt.Errorf("commented_posts.json is corrupt: %w", err)
				}
			}
			if idx.SchemaVersion == 0 {
				idx.SchemaVersion = 1
			}
			if idx.Posts == nil {
				idx.Posts = map[string]seenPost{}
			}
			return nil
		},
		func() ([]byte, error) { return json.MarshalIndent(idx, "", "  ") },
		func() error { return fn(idx) },
	)
	if err != nil {
		return nil, err
	}
	return idx, nil
}

// ---- capacity --------------------------------------------------------------

// commentCapacityPerDay: how many comments this campaign can actually publish in
// a day, from the same two caps the publish side is bounded by. One account
// speaks for a campaign, so its per-day ceiling is (groups it may touch) ×
// (comments per group), and it can never touch more groups in a day than
// comment_groups_per_account_per_day allows.
func commentCapacityPerDay(cfg map[string]any, s systemSettings) (int, string) {
	groups := len(mList(mMap(cfg, "audience"), "groups"))
	if groups == 0 {
		return 0, "no groups chosen for this campaign"
	}
	usable := groups
	if s.CommentGroupsPerAccountPerDay > 0 && usable > s.CommentGroupsPerAccountPerDay {
		usable = s.CommentGroupsPerAccountPerDay
	}
	per := s.CommentsPerGroupPerDay
	if per <= 0 {
		per = 1
	}
	cap := usable * per
	reason := fmt.Sprintf("%d group(s) × %d comment/group/day", usable, per)
	if groups > usable {
		reason += fmt.Sprintf(" (of %d groups; one account may touch %d a day)", groups, s.CommentGroupsPerAccountPerDay)
	}
	return cap, reason
}

// aliveChannelDrafts counts drafts that still stand between the operator and a
// publish — anything awaiting his decision or already approved and waiting for a
// slot. Rejected and held drafts are rewritten in place rather than moved, so
// counting files instead of statuses would permanently block drafting after the
// first cleanup.
func (c *crmStore) aliveChannelDrafts(campaignSlug string) (int, error) {
	cd, err := c.campaignDir(campaignSlug)
	if err != nil {
		return 0, err
	}
	alive := 0
	for _, sub := range []string{"pending_approval", "approved"} {
		_ = filepath.WalkDir(filepath.Join(cd, "outbox", sub), func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
				return nil
			}
			rec, rerr := readJSONFile(p)
			if rerr != nil {
				return nil
			}
			if st := mStr(rec, "status"); st == "pending_approval" || st == "approved" {
				alive++
			}
			return nil
		})
	}
	return alive, nil
}

// ---- the writer ------------------------------------------------------------

// channelDraftWrite deposits ONE comment draft, refusing with a named reason
// rather than writing something the publish side would later have to reject.
// Every gate here is a gate the hand-written route had none of.
func (c *crmStore) channelDraftWrite(campaignSlug string, a channelDraftArgs, s systemSettings) (map[string]any, error) {
	cfg := c.getCampaign(campaignSlug)
	if cfg == nil {
		return nil, storageErrf("campaign %q not found", campaignSlug)
	}
	ch := mStr(cfg, "channel_strategy")
	if ch != "comment" && ch != "post" {
		return nil, storageErrf("channel_mismatch: %q is a %q campaign — this writer only drafts for group-targeted channels", campaignSlug, ch)
	}
	if mStr(cfg, "status") == "paused" {
		return nil, storageErrf("campaign_paused: drafting is paused for %q — resume it on the Campaigns page", campaignSlug)
	}
	body := strings.TrimSpace(a.BodyText)
	if body == "" {
		return nil, storageErrf("empty_body: a comment draft must carry the text to post")
	}

	// The group must be one the operator chose. audience.groups is his list, and
	// it is the only thing that says where this campaign may speak.
	groupID := groupIDFromURL(a.GroupURL)
	if groupID == "" {
		return nil, storageErrf("bad_group_url: %q is not a facebook group url", a.GroupURL)
	}
	allowed := false
	for _, g := range mList(mMap(cfg, "audience"), "groups") {
		if groupIDFromURL(sprint(g)) == groupID {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, storageErrf("group_not_in_campaign: %s is not in this campaign's groups — add it on the campaign page first", a.GroupURL)
	}

	// A COMMENT answers one post, so the post must be pinnable and must live in
	// THAT group or the publish side cannot prove it wrote where it meant to. A
	// group POST answers nothing — the group itself is the target, and the key
	// that stops a repeat is the text, not a permalink.
	var dedupeKey string
	if ch == "comment" {
		postID := canonicalPostID(a.PostURL)
		if postID == "" {
			return nil, storageErrf("unpinnable_post: no post id could be read from %q — a comment needs a permalink", a.PostURL)
		}
		if gid := groupIDFromURL(a.PostURL); gid != groupID {
			return nil, storageErrf("post_outside_group: %s does not belong to group %s", a.PostURL, groupID)
		}
		dedupeKey = postID
	} else {
		dedupeKey = "body:" + bodyFingerprint(body)
	}

	cd, err := c.campaignDir(campaignSlug)
	if err != nil {
		return nil, err
	}

	// Capacity, not daily_quota. Drafting runs at most two days ahead of what
	// this campaign can actually publish.
	capPerDay, capReason := channelCapacityPerDay(ch, cfg, s)
	if capPerDay <= 0 {
		return nil, storageErrf("no_capacity: %s — nothing can be published, so nothing is drafted", capReason)
	}
	ceiling := capPerDay * channelDraftHorizonDays
	alive, err := c.aliveChannelDrafts(campaignSlug)
	if err != nil {
		return nil, err
	}
	if alive >= ceiling {
		return nil, storageErrf("capacity_horizon_full: %d draft(s) already waiting, ceiling %d (%s × %d days) — approve or reject before drafting more",
			alive, ceiling, capReason, channelDraftHorizonDays)
	}

	// One comment per post, ever. Checked and claimed under the same lock as the
	// write, so two drafting passes cannot both take the same post.
	did := newULID("draft_")
	now := nowISO()
	var dup *seenPost
	if _, serr := withSeenPosts(cd, func(idx *seenPostIndex) error {
		if prev, ok := idx.Posts[dedupeKey]; ok {
			p := prev
			dup = &p
			return nil
		}
		idx.Posts[dedupeKey] = seenPost{PostURL: a.PostURL, GroupURL: a.GroupURL, Author: a.PostAuthor,
			DraftID: did, Status: "drafted", FirstAt: now}
		return nil
	}); serr != nil {
		return nil, serr
	}
	if dup != nil {
		what := "a comment for that post"
		if ch == "post" {
			what = "this same text"
		}
		return nil, storageErrf("already_drafted: this campaign already wrote %s on %s (draft %s, %s)",
			what, dup.FirstAt, dup.DraftID, dup.Status)
	}

	seenAt := strings.TrimSpace(a.PostSeenAt)
	if seenAt == "" {
		seenAt = now
	}
	draft := map[string]any{
		"id": did, "schema_version": 1, "created_at": now, "updated_at": now,
		// No lead_id: a commented author never becomes a contact (operator ruling
		// 2026-08-17). `to` carries the post, which is what is being addressed.
		"lead_id": "", "campaign_slug": campaignSlug, "channel": ch, "step": 1,
		"to": strOr(a.PostURL, a.GroupURL), "post_url": a.PostURL, "group_url": a.GroupURL,
		"post_author": a.PostAuthor, "post_excerpt": a.PostExcerpt, "post_seen_at": seenAt,
		"subject": "", "body_text": body, "body_html": "",
		"confidence_band": "review_carefully", "hooks_used": channelHooks(ch, a),
		"tracking": "plain_text", "warnings": channelWarnings(ch),
		"guessed_approved": false, "is_reply": false, "bank_messages_used": []any{},
		"companion_url": "",
		"status":        "pending_approval", "decided_at": "", "decided_by": "",
		"reject_reason": "", "blocker": "", "blocked_at": "",
	}
	dir := filepath.Join(cd, "outbox", "pending_approval", todayStr(now))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	draftPath := filepath.Join(dir, did+".json")
	if err := atomicWriteFile(draftPath, marshalIndentJSON(draft)); err != nil {
		// The post was claimed in the index a moment ago; hand it back rather than
		// burning it on a draft that does not exist.
		_, _ = withSeenPosts(cd, func(idx *seenPostIndex) error {
			delete(idx.Posts, dedupeKey)
			return nil
		})
		return nil, err
	}
	return map[string]any{"ok": true, "draft_id": did, "path": draftPath, "campaign": campaignSlug,
		"channel": ch, "dedupe_key": dedupeKey, "post_url": a.PostURL, "group_url": a.GroupURL,
		"alive_after": alive + 1, "ceiling": ceiling, "capacity_per_day": capPerDay,
		"capacity_reason": capReason, "status": "pending_approval"}, nil
}

// bodyFingerprint: whitespace- and case-insensitive digest of a post body. A
// group post has no permalink to dedupe on, so the text is the identity — the
// thing that must never be published twice is the same words, not the same url.
func bodyFingerprint(body string) string {
	norm := strings.Join(strings.Fields(strings.ToLower(body)), " ")
	sum := sha1.Sum([]byte(norm))
	return hex.EncodeToString(sum[:])[:16]
}

// channelCapacityPerDay: what this campaign can actually publish in a day.
// Comments are bounded by group diversity; a group POST is bounded per account,
// because it is one standalone piece of content in front of a whole group and
// the exposure does not divide by how many groups the operator listed.
func channelCapacityPerDay(channel string, cfg map[string]any, s systemSettings) (int, string) {
	if channel != "post" {
		return commentCapacityPerDay(cfg, s)
	}
	if len(mList(mMap(cfg, "audience"), "groups")) == 0 {
		return 0, "no groups chosen for this campaign"
	}
	n := s.PostsPerAccountPerDay
	if n <= 0 {
		n = 2
	}
	return n, fmt.Sprintf("%d new post(s)/account/day", n)
}

func channelHooks(channel string, a channelDraftArgs) []any {
	if channel == "post" {
		// A group post answers no single post; what it is grounded in is the reading
		// the agent did, recorded as the excerpt it worked from.
		if strings.TrimSpace(a.PostExcerpt) == "" {
			return []any{}
		}
		return []any{map[string]any{"type": "group_reading", "evidence_url": a.GroupURL}}
	}
	return []any{map[string]any{"type": "social_post", "evidence_url": a.PostURL}}
}

func channelWarnings(channel string) []any {
	if channel == "post" {
		return []any{"facebook_group_post", "most_exposed_action", "may_await_group_admin"}
	}
	return []any{"facebook_public_comment", "one_comment_per_group_per_day"}
}
