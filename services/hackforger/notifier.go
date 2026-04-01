// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	issues_model "forgejo.org/models/issues"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"
	notify_service "forgejo.org/services/notify"
)

type hackforgerNotifier struct {
	notify_service.NullNotifier
}

var _ notify_service.Notifier = &hackforgerNotifier{}

// Init registers the HackForger notifier.
func Init() error {
	notify_service.RegisterNotifier(&hackforgerNotifier{})
	return nil
}

// AudienceType determines how feed events are distributed.
// Values are bit flags and may be combined with bitwise OR.
type AudienceType int

const (
	AudienceGlobal       AudienceType = 1 << iota // 1 — UserID=0, visible to everyone
	AudienceFollowers                              // 2 — Visible to ActUser's followers
	AudienceOrgMembers                             // 4 — Visible to org members
	AudienceRepoWatchers                           // 8 — Visible to repo watchers
	AudienceDirectUser                             // 16 — Visible to a single target user
)

// HackforgerActionOpts holds the parameters for publishing a HackForger feed event.
type HackforgerActionOpts struct {
	ActUserID    int64
	OpType       activities_model.ActionType
	EntityType   string
	EntityID     int64
	EntityName   string
	EntitySlug   string
	RepoID       int64
	Content      any
	AudienceType AudienceType
	OrgID        int64 // used when AudienceRepoWatchers or AudienceOrgMembers flag is set
	TargetUserID int64 // used when AudienceDirectUser flag is set
}

// PublishHackforgerAction writes HackForger events to the hackforger_action table.
// Unlike Forgejo's NotifyWatchers (which only handles repo watchers),
// this supports 4 audience strategies as bit flags per the spec.
//
// Always inserts an actor record. Additional records are inserted for each
// enabled audience flag: global (user_id=0), followers, org members, repo watchers.
func PublishHackforgerAction(ctx context.Context, opts *HackforgerActionOpts) error {
	contentBytes, err := json.Marshal(opts.Content)
	if err != nil {
		return err
	}
	contentStr := string(contentBytes)
	now := timeutil.TimeStampNow()

	// inserted tracks which user_ids have already received a feed record for
	// this event, preventing duplicate rows when a user appears in multiple
	// audience sets (e.g., both follower and repo watcher).
	inserted := make(map[int64]bool)

	insert := func(userID int64) error {
		if inserted[userID] {
			return nil
		}
		inserted[userID] = true
		rec := &hackforger_model.HackforgerAction{
			UserID:      userID,
			ActUserID:   opts.ActUserID,
			OpType:      opts.OpType,
			EntityType:  opts.EntityType,
			EntityID:    opts.EntityID,
			EntityName:  opts.EntityName,
			EntitySlug:  opts.EntitySlug,
			OrgID:       opts.OrgID,
			RepoID:      opts.RepoID,
			Content:     contentStr,
			CreatedUnix: now,
		}
		if _, err := db.GetEngine(ctx).Insert(rec); err != nil {
			log.Error("PublishHackforgerAction (user_id=%d): %v", userID, err)
			return err
		}
		return nil
	}

	// Always insert the actor's own feed record.
	if err := insert(opts.ActUserID); err != nil {
		return err
	}

	// Global flag: insert a user_id=0 record visible to everyone.
	if opts.AudienceType&AudienceGlobal != 0 {
		if err := insert(0); err != nil {
			return err
		}
	}

	// Followers flag: insert for each follower of the acting user.
	if opts.AudienceType&AudienceFollowers != 0 {
		var followerIDs []int64
		if err := db.GetEngine(ctx).Table("follow").
			Where("follow_id = ?", opts.ActUserID).
			Cols("user_id").Find(&followerIDs); err != nil {
			log.Error("PublishHackforgerAction (followers query): %v", err)
		} else {
			for _, uid := range followerIDs {
				if uid == opts.ActUserID {
					continue
				}
				_ = insert(uid)
			}
		}
	}

	// OrgMembers flag: insert for each member of the target org.
	if opts.AudienceType&AudienceOrgMembers != 0 {
		if opts.OrgID > 0 {
			var memberIDs []int64
			if err := db.GetEngine(ctx).Table("org_user").
				Where("org_id = ?", opts.OrgID).
				Cols("uid").Find(&memberIDs); err != nil {
				log.Error("PublishHackforgerAction (org members query): %v", err)
			} else {
				for _, uid := range memberIDs {
					if uid == opts.ActUserID {
						continue
					}
					_ = insert(uid)
				}
			}
		}
	}

	// RepoWatchers flag: insert for each watcher of the target repo.
	if opts.AudienceType&AudienceRepoWatchers != 0 {
		if opts.RepoID > 0 {
			var watcherIDs []int64
			if err := db.GetEngine(ctx).Table("watch").
				Where("repo_id = ? AND mode != 2", opts.RepoID).
				Cols("user_id").Find(&watcherIDs); err != nil {
				log.Error("PublishHackforgerAction (watchers query): %v", err)
			} else {
				for _, uid := range watcherIDs {
					if uid == opts.ActUserID {
						continue
					}
					_ = insert(uid)
				}
			}
		}
	}

	// DirectUser flag: insert for a specific target user.
	if opts.AudienceType&AudienceDirectUser != 0 {
		if opts.TargetUserID > 0 {
			_ = insert(opts.TargetUserID)
		}
	}

	return nil
}

// MergePullRequest is called when a PR is merged. We check if the PR's
// issue has a linked Bounty and trigger status transitions if so.
// This is a skeleton — full logic added in Phase 1.
func (n *hackforgerNotifier) MergePullRequest(ctx context.Context, doer *user_model.User, pr *issues_model.PullRequest) {
	// Phase 1: Check if PR's Issue has a Bounty, and if doer is the Claimer,
	// transition to InReview status.
	_ = doer
	_ = pr
	_ = ctx
	_ = hackforger_model.ActionBountyDelivered // ensure import is used
}
