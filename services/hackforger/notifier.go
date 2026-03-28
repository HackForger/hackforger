// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
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

// AudienceType is a bitmask determining how feed events are distributed.
// Multiple audiences can be combined with bitwise OR.
type AudienceType int

const (
	AudienceGlobal       AudienceType = 1 << 0 // 1 — UserID=0, visible to everyone
	AudienceFollowers    AudienceType = 1 << 1 // 2 — Visible to ActUser's followers
	AudienceOrgMembers   AudienceType = 1 << 2 // 4 — Visible to org members
	AudienceRepoWatchers AudienceType = 1 << 3 // 8 — Visible to repo watchers
)

// HackforgerActionOpts holds the parameters for publishing a HackForger feed event.
type HackforgerActionOpts struct {
	ActUserID    int64
	OpType       activities_model.ActionType
	RepoID       int64
	Content      any
	AudienceType AudienceType
	OrgID        int64 // used when AudienceType includes AudienceOrgMembers
}

// PublishHackforgerAction writes HackForger events to the action table.
// Unlike Forgejo's NotifyWatchers (which only handles repo watchers),
// this supports 4 audience strategies via bitmask composition.
func PublishHackforgerAction(ctx context.Context, opts *HackforgerActionOpts) error {
	contentBytes, err := json.Marshal(opts.Content)
	if err != nil {
		return err
	}
	contentStr := string(contentBytes)
	now := timeutil.TimeStampNow()

	// Always insert the actor's own action record.
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

	// Collect all target user IDs (dedup via map, excluding actor).
	targets := make(map[int64]bool)

	// Global: insert a UserID=0 record (visible on explore feed).
	if opts.AudienceType&AudienceGlobal != 0 {
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

	// Followers: query the `follow` table.
	// follow.user_id = follower, follow.follow_id = the person being followed.
	if opts.AudienceType&AudienceFollowers != 0 {
		var followerIDs []int64
		if err := db.GetEngine(ctx).Table("follow").
			Where("follow_id = ?", opts.ActUserID).
			Cols("user_id").
			Find(&followerIDs); err != nil {
			log.Error("PublishHackforgerAction (followers query): %v", err)
			return err
		}
		for _, id := range followerIDs {
			targets[id] = true
		}
	}

	// Org members: query the `org_user` table.
	if opts.AudienceType&AudienceOrgMembers != 0 && opts.OrgID > 0 {
		var memberIDs []int64
		if err := db.GetEngine(ctx).Table("org_user").
			Where("org_id = ?", opts.OrgID).
			Cols("uid").
			Find(&memberIDs); err != nil {
			log.Error("PublishHackforgerAction (org members query): %v", err)
			return err
		}
		for _, id := range memberIDs {
			targets[id] = true
		}
	}

	// Repo watchers: query the `watch` table, excluding WatchModeDont (2).
	if opts.AudienceType&AudienceRepoWatchers != 0 && opts.RepoID > 0 {
		var watcherIDs []int64
		if err := db.GetEngine(ctx).Table("watch").
			Where("repo_id = ? AND mode <> ?", opts.RepoID, repo_model.WatchModeDont).
			Cols("user_id").
			Find(&watcherIDs); err != nil {
			log.Error("PublishHackforgerAction (repo watchers query): %v", err)
			return err
		}
		for _, id := range watcherIDs {
			targets[id] = true
		}
	}

	// Remove the actor (already inserted above).
	delete(targets, opts.ActUserID)

	// Batch insert action records for all resolved targets.
	if len(targets) > 0 {
		actions := make([]*activities_model.Action, 0, len(targets))
		for uid := range targets {
			actions = append(actions, &activities_model.Action{
				ActUserID:   opts.ActUserID,
				UserID:      uid,
				OpType:      opts.OpType,
				RepoID:      opts.RepoID,
				Content:     contentStr,
				CreatedUnix: now,
			})
		}
		if _, err := db.GetEngine(ctx).Insert(actions); err != nil {
			log.Error("PublishHackforgerAction (batch targets): %v", err)
			return err
		}
	}

	return nil
}

// MergePullRequest is called when a PR is merged. If the PR's issue has a
// linked Bounty in Claimed status and the merger is the claimer, transition
// the bounty to InReview and publish a BountyDelivered feed event.
func (n *hackforgerNotifier) MergePullRequest(ctx context.Context, doer *user_model.User, pr *issues_model.PullRequest) {
	if err := pr.LoadIssue(ctx); err != nil {
		log.Error("hackforgerNotifier.MergePullRequest: LoadIssue: %v", err)
		return
	}

	bounty, err := hackforger_model.GetBountyByIssueID(ctx, pr.Issue.ID)
	if err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			return // No bounty linked to this issue — nothing to do.
		}
		log.Error("hackforgerNotifier.MergePullRequest: GetBountyByIssueID: %v", err)
		return
	}

	// Only transition if the bounty is Claimed and the merger is the claimer.
	if bounty.Status != hackforger_model.BountyStatusClaimed || doer.ID != bounty.ClaimerID {
		return
	}

	bounty.Status = hackforger_model.BountyStatusInReview
	if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
		log.Error("hackforgerNotifier.MergePullRequest: UpdateBounty: %v", err)
		return
	}

	if err := PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doer.ID,
		OpType:       hackforger_model.ActionBountyDelivered,
		RepoID:       bounty.RepoID,
		AudienceType: AudienceGlobal | AudienceRepoWatchers,
		Content: &hackforger_model.HackforgerPhaseContent{
			HackforgerActionContent: hackforger_model.HackforgerActionContent{
				EntityType: "bounty",
				EntityID:   bounty.ID,
				EntityName: bounty.Title,
			},
			OldStatus: "claimed",
			NewStatus: "in_review",
		},
	}); err != nil {
		log.Error("hackforgerNotifier.MergePullRequest: PublishHackforgerAction: %v", err)
	}
}
