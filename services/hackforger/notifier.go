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
type AudienceType int

const (
	AudienceGlobal       AudienceType = iota // UserID=0, visible to everyone
	AudienceFollowers                        // Visible to ActUser's followers
	AudienceOrgMembers                       // Visible to org members
	AudienceRepoWatchers                     // Visible to repo watchers
)

// HackforgerActionOpts holds the parameters for publishing a HackForger feed event.
type HackforgerActionOpts struct {
	ActUserID    int64
	OpType       activities_model.ActionType
	RepoID       int64
	Content      any
	AudienceType AudienceType
	OrgID        int64 // used when AudienceType == AudienceOrgMembers
}

// PublishHackforgerAction writes HackForger events to the action table.
// Unlike Forgejo's NotifyWatchers (which only handles repo watchers),
// this supports 4 audience strategies per the spec.
//
// P0 skeleton: inserts for actor + global (UserID=0) only.
// Phase 1: adds full audience resolution (followers, org members, repo watchers).
func PublishHackforgerAction(ctx context.Context, opts *HackforgerActionOpts) error {
	contentBytes, err := json.Marshal(opts.Content)
	if err != nil {
		return err
	}
	contentStr := string(contentBytes)
	now := timeutil.TimeStampNow()

	// Always insert the actor's own action record
	actorAction := &activities_model.Action{
		ActUserID:   opts.ActUserID,
		UserID:      opts.ActUserID,
		OpType:      opts.OpType,
		RepoID:      opts.RepoID,
		Content:     contentStr,
		CreatedUnix: now,
	}
	if _, err := db.GetEngine(ctx).Insert(actorAction); err != nil {
		log.Error("PublishHackforgerAction (actor): %v", err)
		return err
	}

	// For global events, also insert a UserID=0 record
	if opts.AudienceType == AudienceGlobal {
		globalAction := &activities_model.Action{
			ActUserID:   opts.ActUserID,
			UserID:      0,
			OpType:      opts.OpType,
			RepoID:      opts.RepoID,
			Content:     contentStr,
			CreatedUnix: now,
		}
		if _, err := db.GetEngine(ctx).Insert(globalAction); err != nil {
			log.Error("PublishHackforgerAction (global): %v", err)
			return err
		}
	}

	// Phase 1: resolve audience based on opts.AudienceType
	switch opts.AudienceType {
	case AudienceFollowers:
		var followerIDs []int64
		err := db.GetEngine(ctx).Table("follow").
			Where("follow_id = ?", opts.ActUserID).
			Cols("user_id").Find(&followerIDs)
		if err != nil {
			log.Error("PublishHackforgerAction (followers query): %v", err)
			break
		}
		for _, uid := range followerIDs {
			if uid == opts.ActUserID {
				continue
			}
			fa := &activities_model.Action{
				ActUserID:   opts.ActUserID,
				UserID:      uid,
				OpType:      opts.OpType,
				RepoID:      opts.RepoID,
				Content:     contentStr,
				CreatedUnix: now,
			}
			if _, err := db.GetEngine(ctx).Insert(fa); err != nil {
				log.Error("PublishHackforgerAction (follower %d): %v", uid, err)
			}
		}

	case AudienceOrgMembers:
		if opts.OrgID == 0 {
			break
		}
		var memberIDs []int64
		err := db.GetEngine(ctx).Table("org_user").
			Where("org_id = ?", opts.OrgID).
			Cols("uid").Find(&memberIDs)
		if err != nil {
			log.Error("PublishHackforgerAction (org members query): %v", err)
			break
		}
		for _, uid := range memberIDs {
			if uid == opts.ActUserID {
				continue
			}
			ma := &activities_model.Action{
				ActUserID:   opts.ActUserID,
				UserID:      uid,
				OpType:      opts.OpType,
				RepoID:      opts.RepoID,
				Content:     contentStr,
				CreatedUnix: now,
			}
			if _, err := db.GetEngine(ctx).Insert(ma); err != nil {
				log.Error("PublishHackforgerAction (org member %d): %v", uid, err)
			}
		}

	case AudienceRepoWatchers:
		if opts.RepoID == 0 {
			break
		}
		var watcherIDs []int64
		err := db.GetEngine(ctx).Table("watch").
			Where("repo_id = ? AND mode != 2", opts.RepoID).
			Cols("user_id").Find(&watcherIDs)
		if err != nil {
			log.Error("PublishHackforgerAction (watchers query): %v", err)
			break
		}
		for _, uid := range watcherIDs {
			if uid == opts.ActUserID {
				continue
			}
			wa := &activities_model.Action{
				ActUserID:   opts.ActUserID,
				UserID:      uid,
				OpType:      opts.OpType,
				RepoID:      opts.RepoID,
				Content:     contentStr,
				CreatedUnix: now,
			}
			if _, err := db.GetEngine(ctx).Insert(wa); err != nil {
				log.Error("PublishHackforgerAction (watcher %d): %v", uid, err)
			}
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
