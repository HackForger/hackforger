// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/timeutil"

	"xorm.io/builder"
)

// HackforgerAction is HackForger's own feed event store, separate from
// Forgejo's repo-centric `action` table. user_id=0 means a global event.
type HackforgerAction struct {
	ID          int64                       `xorm:"pk autoincr"`
	UserID      int64                       `xorm:"NOT NULL INDEX(idx_hfa_user_created)"`
	ActUserID   int64                       `xorm:"NOT NULL INDEX(idx_hfa_actuser_created)"`
	OpType      activities_model.ActionType `xorm:"NOT NULL"`
	EntityType  string                      `xorm:"VARCHAR(20) NOT NULL INDEX(idx_hfa_entity)"`
	EntityID    int64                       `xorm:"NOT NULL DEFAULT 0"`
	EntityName  string                      `xorm:"VARCHAR(255)"`
	EntitySlug  string                      `xorm:"VARCHAR(255)"`
	OrgID       int64                       `xorm:"DEFAULT 0"`
	RepoID      int64                       `xorm:"DEFAULT 0"`
	Content     string                      `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp          `xorm:"INDEX(idx_hfa_user_created) INDEX(idx_hfa_actuser_created) INDEX(idx_hfa_entity) created"`

	ActUser *user_model.User `xorm:"-"`
}

func init() {
	db.RegisterModel(new(HackforgerAction))
}

// GetHackforgerFeedsOptions holds filter and pagination parameters for
// querying HackforgerAction rows.
type GetHackforgerFeedsOptions struct {
	db.ListOptions
	UserID        int64
	ActUserID     int64
	EntityType    string
	EntityID      int64
	IncludeGlobal bool
}

// GetHackforgerFeeds returns a paginated list of HackforgerAction rows and
// their total count, filtered by the provided options.
func GetHackforgerFeeds(ctx context.Context, opts GetHackforgerFeedsOptions) ([]*HackforgerAction, int64, error) {
	cond := builder.NewCond()

	if opts.UserID > 0 {
		if opts.IncludeGlobal {
			cond = cond.And(builder.Or(
				builder.Eq{"user_id": opts.UserID},
				builder.Eq{"user_id": 0},
			))
		} else {
			cond = cond.And(builder.Eq{"user_id": opts.UserID})
		}
	} else if opts.IncludeGlobal {
		cond = cond.And(builder.Eq{"user_id": 0})
	}

	if opts.ActUserID > 0 {
		cond = cond.And(builder.Eq{"act_user_id": opts.ActUserID})
	}

	if opts.EntityType != "" {
		cond = cond.And(builder.Eq{"entity_type": opts.EntityType})
	}

	if opts.EntityID > 0 {
		cond = cond.And(builder.Eq{"entity_id": opts.EntityID})
	}

	opts.SetDefaultValues()
	sess := db.GetEngine(ctx).Where(cond)

	// Deduplicate: an actor may have both a personal record (UserID=actorID)
	// and a global record (UserID=0) for the same event. Group to avoid dupes.
	if opts.IncludeGlobal && opts.UserID > 0 {
		sess = sess.GroupBy("act_user_id, op_type, entity_type, entity_id, created_unix")
	}

	sess = db.SetSessionPagination(sess, &opts.ListOptions)

	actions := make([]*HackforgerAction, 0, opts.PageSize)
	count, err := sess.Desc("created_unix").FindAndCount(&actions)
	return actions, count, err
}

// GetEntityTimeline returns a deduplicated timeline of events for a single
// entity (e.g. a hackathon or bounty), grouped to suppress duplicate entries
// from the same actor performing the same action at the same second.
func GetEntityTimeline(ctx context.Context, entityType string, entityID int64, opts db.ListOptions) ([]*HackforgerAction, int64, error) {
	cond := builder.Eq{"entity_type": entityType, "entity_id": entityID}

	opts.SetDefaultValues()
	sess := db.GetEngine(ctx).Where(cond).
		GroupBy("act_user_id, op_type, entity_type, entity_id, created_unix")
	sess = db.SetSessionPagination(sess, &opts)

	actions := make([]*HackforgerAction, 0, opts.PageSize)
	count, err := sess.Desc("created_unix").FindAndCount(&actions)
	return actions, count, err
}

// LoadActUsers batch-loads the acting user records for a slice of actions and
// sets each action's ActUser field. Missing users are left nil.
func LoadActUsers(ctx context.Context, actions []*HackforgerAction) error {
	if len(actions) == 0 {
		return nil
	}

	userIDs := make([]int64, 0, len(actions))
	seen := make(map[int64]bool)
	for _, a := range actions {
		if !seen[a.ActUserID] {
			userIDs = append(userIDs, a.ActUserID)
			seen[a.ActUserID] = true
		}
	}

	userMap := make(map[int64]*user_model.User)
	if len(userIDs) > 0 {
		var users []*user_model.User
		if err := db.GetEngine(ctx).In("id", userIDs).Find(&users); err != nil {
			return err
		}
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	for _, a := range actions {
		a.ActUser = userMap[a.ActUserID]
	}
	return nil
}
