// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"testing"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishHackforgerAction_Global(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := PublishHackforgerAction(db.DefaultContext, &HackforgerActionOpts{
		ActUserID:    2,
		OpType:       hackforger_model.ActionBountyCreated,
		RepoID:       1,
		AudienceType: AudienceGlobal,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "bounty",
			EntityID:   1,
			EntityName: "Test Bounty",
		},
	})
	require.NoError(t, err)

	// Verify global record (UserID=0) was created.
	var globalActions []*activities_model.Action
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND user_id = 0", hackforger_model.ActionBountyCreated).
		Find(&globalActions)
	require.NoError(t, err)
	assert.NotEmpty(t, globalActions, "expected a global action record with UserID=0")
	assert.Equal(t, int64(2), globalActions[0].ActUserID)

	// Verify actor's own record was created.
	var actorActions []*activities_model.Action
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND user_id = 2 AND act_user_id = 2", hackforger_model.ActionBountyCreated).
		Find(&actorActions)
	require.NoError(t, err)
	assert.NotEmpty(t, actorActions, "expected actor's own action record")
}

func TestPublishHackforgerAction_Followers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// User 2 has followers: user 4 and user 8 (per follow.yml fixture).
	err := PublishHackforgerAction(db.DefaultContext, &HackforgerActionOpts{
		ActUserID:    2,
		OpType:       hackforger_model.ActionBountyClaimed,
		RepoID:       1,
		AudienceType: AudienceFollowers,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "bounty",
			EntityID:   2,
			EntityName: "Follower Bounty",
		},
	})
	require.NoError(t, err)

	// Verify actor record exists.
	var actorActions []*activities_model.Action
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND user_id = 2 AND act_user_id = 2", hackforger_model.ActionBountyClaimed).
		Find(&actorActions)
	require.NoError(t, err)
	assert.Len(t, actorActions, 1, "expected actor's own action record")

	// Verify follower records exist (users 4 and 8).
	var followerActions []*activities_model.Action
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND act_user_id = 2 AND user_id != 2", hackforger_model.ActionBountyClaimed).
		Find(&followerActions)
	require.NoError(t, err)
	assert.Len(t, followerActions, 2, "expected action records for 2 followers")

	followerIDs := make(map[int64]bool)
	for _, a := range followerActions {
		followerIDs[a.UserID] = true
	}
	assert.True(t, followerIDs[4], "expected action for follower user 4")
	assert.True(t, followerIDs[8], "expected action for follower user 8")
}

func TestPublishHackforgerAction_CombinedAudience(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Use Global | RepoWatchers for user 2 on repo 1.
	// Repo 1 watchers (excluding mode=2): users 1, 4, 9, 11.
	// Actor is user 2, so all 4 watchers should get records.
	err := PublishHackforgerAction(db.DefaultContext, &HackforgerActionOpts{
		ActUserID:    2,
		OpType:       hackforger_model.ActionBountyCompleted,
		RepoID:       1,
		AudienceType: AudienceGlobal | AudienceRepoWatchers,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "bounty",
			EntityID:   3,
			EntityName: "Combined Bounty",
		},
	})
	require.NoError(t, err)

	// Verify global record.
	var globalActions []*activities_model.Action
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND user_id = 0", hackforger_model.ActionBountyCompleted).
		Find(&globalActions)
	require.NoError(t, err)
	assert.Len(t, globalActions, 1, "expected one global action record")

	// Verify watcher records were created (users 1, 4, 9, 11 — not user 8 who has mode=2).
	var allActions []*activities_model.Action
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND act_user_id = 2", hackforger_model.ActionBountyCompleted).
		Find(&allActions)
	require.NoError(t, err)

	userIDs := make(map[int64]bool)
	for _, a := range allActions {
		userIDs[a.UserID] = true
	}
	// Should have: actor(2), global(0), watchers(1, 4, 9, 11)
	assert.True(t, userIDs[0], "expected global record")
	assert.True(t, userIDs[2], "expected actor record")
	assert.True(t, userIDs[1], "expected watcher user 1")
	assert.True(t, userIDs[4], "expected watcher user 4")
	assert.True(t, userIDs[9], "expected watcher user 9")
	assert.True(t, userIDs[11], "expected watcher user 11")
	assert.False(t, userIDs[8], "user 8 has WatchModeDont, should not receive action")
}

func TestPublishHackforgerAction_Dedup(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// User 2's followers: users 4, 8.
	// Repo 1 watchers (excluding mode=2): users 1, 4, 9, 11.
	// User 4 appears in BOTH followers and watchers — should only get one record.
	err := PublishHackforgerAction(db.DefaultContext, &HackforgerActionOpts{
		ActUserID:    2,
		OpType:       hackforger_model.ActionBountyDelivered,
		RepoID:       1,
		AudienceType: AudienceFollowers | AudienceRepoWatchers,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "bounty",
			EntityID:   4,
			EntityName: "Dedup Bounty",
		},
	})
	require.NoError(t, err)

	// Count records for user 4 — should be exactly 1 (dedup).
	var user4Actions []*activities_model.Action
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND user_id = 4 AND act_user_id = 2", hackforger_model.ActionBountyDelivered).
		Find(&user4Actions)
	require.NoError(t, err)
	assert.Len(t, user4Actions, 1, "user 4 should appear exactly once despite being in both followers and watchers")

	// Verify the complete set of unique target users:
	// Followers(4, 8) + Watchers(1, 4, 9, 11) - Actor(2) = {1, 4, 8, 9, 11}
	// Plus actor record = 6 total records.
	var allActions []*activities_model.Action
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND act_user_id = 2", hackforger_model.ActionBountyDelivered).
		Find(&allActions)
	require.NoError(t, err)
	assert.Len(t, allActions, 6, "expected 6 total action records (actor + 5 unique targets)")
}
