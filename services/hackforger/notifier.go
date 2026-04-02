// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

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

// publishFeed writes a HackForger event to the hackforger_action table.
// It supports multiple audience strategies as bit flags.
func publishFeed(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	contentBytes, err := json.Marshal(opts.Content)
	if err != nil {
		log.Error("publishFeed: marshal content: %v", err)
		return
	}
	contentStr := string(contentBytes)
	now := timeutil.TimeStampNow()

	// inserted tracks which user_ids have already received a feed record for
	// this event, preventing duplicate rows when a user appears in multiple
	// audience sets (e.g., both follower and repo watcher).
	inserted := make(map[int64]bool)

	insert := func(userID int64) {
		if inserted[userID] {
			return
		}
		inserted[userID] = true
		rec := &hackforger_model.HackforgerAction{
			UserID:      userID,
			ActUserID:   doer.ID,
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
			log.Error("publishFeed (user_id=%d): %v", userID, err)
		}
	}

	// Always insert the actor's own feed record.
	insert(doer.ID)

	// Global flag: insert a user_id=0 record visible to everyone.
	if opts.AudienceType&notify_service.AudienceGlobal != 0 {
		insert(0)
	}

	// Followers flag: insert for each follower of the acting user.
	if opts.AudienceType&notify_service.AudienceFollowers != 0 {
		var followerIDs []int64
		if err := db.GetEngine(ctx).Table("follow").
			Where("follow_id = ?", doer.ID).
			Cols("user_id").Find(&followerIDs); err != nil {
			log.Error("publishFeed (followers query): %v", err)
		} else {
			for _, uid := range followerIDs {
				if uid == doer.ID {
					continue
				}
				insert(uid)
			}
		}
	}

	// OrgMembers flag: insert for each member of the target org.
	if opts.AudienceType&notify_service.AudienceOrgMembers != 0 {
		if opts.OrgID > 0 {
			var memberIDs []int64
			if err := db.GetEngine(ctx).Table("org_user").
				Where("org_id = ?", opts.OrgID).
				Cols("uid").Find(&memberIDs); err != nil {
				log.Error("publishFeed (org members query): %v", err)
			} else {
				for _, uid := range memberIDs {
					if uid == doer.ID {
						continue
					}
					insert(uid)
				}
			}
		}
	}

	// RepoWatchers flag: insert for each watcher of the target repo.
	if opts.AudienceType&notify_service.AudienceRepoWatchers != 0 {
		if opts.RepoID > 0 {
			var watcherIDs []int64
			if err := db.GetEngine(ctx).Table("watch").
				Where("repo_id = ? AND mode != 2", opts.RepoID).
				Cols("user_id").Find(&watcherIDs); err != nil {
				log.Error("publishFeed (watchers query): %v", err)
			} else {
				for _, uid := range watcherIDs {
					if uid == doer.ID {
						continue
					}
					insert(uid)
				}
			}
		}
	}

	// DirectUser flag: insert for a specific target user.
	if opts.AudienceType&notify_service.AudienceDirectUser != 0 {
		if opts.TargetUserID > 0 {
			insert(opts.TargetUserID)
		}
	}
}

// HackforgerEntityCreated handles creation events (hackathon created, bounty created, etc.).
func (n *hackforgerNotifier) HackforgerEntityCreated(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	publishFeed(ctx, doer, opts)
}

// HackforgerEntityUpdated handles update events (name/description changes).
func (n *hackforgerNotifier) HackforgerEntityUpdated(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	if opts.OpType > 0 {
		publishFeed(ctx, doer, opts)
	}
}

// HackforgerEntityDeleted handles deletion events (no-op: entity is gone).
func (n *hackforgerNotifier) HackforgerEntityDeleted(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	// No-op: entity is gone, no feed record needed.
}

// HackforgerEntityStatusChanged handles status transition events.
func (n *hackforgerNotifier) HackforgerEntityStatusChanged(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	publishFeed(ctx, doer, opts)
}

// MergePullRequest is called when a PR is merged. We check if the PR's
// issue has a linked Bounty and trigger status transitions if so.
// This is a skeleton -- full logic added in Phase 1.
func (n *hackforgerNotifier) MergePullRequest(ctx context.Context, doer *user_model.User, pr *issues_model.PullRequest) {
	// Phase 1: Check if PR's Issue has a Bounty, and if doer is the Claimer,
	// transition to InReview status.
	_ = doer
	_ = pr
	_ = ctx
	_ = hackforger_model.ActionBountyDelivered // ensure import is used
}
