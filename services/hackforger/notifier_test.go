// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"sync"
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	user_model "forgejo.org/models/user"
	"forgejo.org/models/unittest"
	notify_service "forgejo.org/services/notify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var notifierOnce sync.Once

func ensureNotifierRegistered() {
	notifierOnce.Do(func() {
		notify_service.RegisterNotifier(&hackforgerNotifier{})
	})
}

func TestPublishFeed_Global(t *testing.T) {
	ensureNotifierRegistered()
	require.NoError(t, unittest.PrepareTestDatabase())

	doer, err := user_model.GetUserByID(db.DefaultContext, 2)
	require.NoError(t, err)

	notify_service.HackforgerEntityCreated(db.DefaultContext, doer, &notify_service.HackforgerEventOpts{
		OpType:       hackforger_model.ActionBountyCreated,
		EntityType:   "bounty",
		EntityID:     1,
		EntityName:   "Test Bounty",
		RepoID:       1,
		AudienceType: notify_service.AudienceGlobal,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "bounty",
			EntityID:   1,
			EntityName: "Test Bounty",
		},
	})

	// Verify global record (UserID=0) was created in hackforger_action table.
	var globalActions []*hackforger_model.HackforgerAction
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND user_id = 0", hackforger_model.ActionBountyCreated).
		Find(&globalActions)
	require.NoError(t, err)
	assert.NotEmpty(t, globalActions, "expected a global action record with UserID=0")
	assert.Equal(t, int64(2), globalActions[0].ActUserID)

	// Verify actor's own record was created in hackforger_action table.
	var actorActions []*hackforger_model.HackforgerAction
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND user_id = 2 AND act_user_id = 2", hackforger_model.ActionBountyCreated).
		Find(&actorActions)
	require.NoError(t, err)
	assert.NotEmpty(t, actorActions, "expected actor's own action record")
}

func TestPublishFeed_Followers(t *testing.T) {
	ensureNotifierRegistered()
	require.NoError(t, unittest.PrepareTestDatabase())

	doer, err := user_model.GetUserByID(db.DefaultContext, 2)
	require.NoError(t, err)

	// User 2 has followers: user 4 and user 8 (per follow.yml fixture).
	notify_service.HackforgerEntityStatusChanged(db.DefaultContext, doer, &notify_service.HackforgerEventOpts{
		OpType:       hackforger_model.ActionBountyClaimed,
		RepoID:       1,
		AudienceType: notify_service.AudienceFollowers,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "bounty",
			EntityID:   2,
			EntityName: "Follower Bounty",
		},
	})

	// Verify actor record exists.
	var actorActions []*hackforger_model.HackforgerAction
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND user_id = 2 AND act_user_id = 2", hackforger_model.ActionBountyClaimed).
		Find(&actorActions)
	require.NoError(t, err)
	assert.Len(t, actorActions, 1, "expected actor's own action record")

	// Verify follower records exist (users 4 and 8).
	var followerActions []*hackforger_model.HackforgerAction
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

func TestPublishFeed_CombinedAudience(t *testing.T) {
	ensureNotifierRegistered()
	require.NoError(t, unittest.PrepareTestDatabase())

	doer, err := user_model.GetUserByID(db.DefaultContext, 2)
	require.NoError(t, err)

	// Test RepoWatchers audience for user 2 on repo 1.
	notify_service.HackforgerEntityStatusChanged(db.DefaultContext, doer, &notify_service.HackforgerEventOpts{
		OpType:       hackforger_model.ActionBountyCompleted,
		EntityType:   "bounty",
		EntityID:     3,
		EntityName:   "Combined Bounty",
		RepoID:       1,
		AudienceType: notify_service.AudienceRepoWatchers,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "bounty",
			EntityID:   3,
			EntityName: "Combined Bounty",
		},
	})

	// Verify action records were created in hackforger_action table.
	var allActions []*hackforger_model.HackforgerAction
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND act_user_id = 2", hackforger_model.ActionBountyCompleted).
		Find(&allActions)
	require.NoError(t, err)

	userIDs := make(map[int64]bool)
	for _, a := range allActions {
		userIDs[a.UserID] = true
	}
	// Actor always gets a record.
	assert.True(t, userIDs[2], "expected actor record")
	// At least one watcher record should exist (repo 1 has watchers in fixtures).
	assert.GreaterOrEqual(t, len(allActions), 2, "expected at least actor + 1 watcher")
}

func TestPublishFeed_Dedup(t *testing.T) {
	ensureNotifierRegistered()
	require.NoError(t, unittest.PrepareTestDatabase())

	doer, err := user_model.GetUserByID(db.DefaultContext, 2)
	require.NoError(t, err)

	// User 2's followers: users 4, 8.
	// Repo 1 watchers (excluding mode=2): users 1, 4, 9, 11.
	// User 4 appears in BOTH followers and watchers -- should only get one record.
	notify_service.HackforgerEntityStatusChanged(db.DefaultContext, doer, &notify_service.HackforgerEventOpts{
		OpType:       hackforger_model.ActionBountyDelivered,
		RepoID:       1,
		AudienceType: notify_service.AudienceFollowers | notify_service.AudienceRepoWatchers,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "bounty",
			EntityID:   4,
			EntityName: "Dedup Bounty",
		},
	})

	// Count records for user 4 -- should be exactly 1 (dedup).
	var user4Actions []*hackforger_model.HackforgerAction
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND user_id = 4 AND act_user_id = 2", hackforger_model.ActionBountyDelivered).
		Find(&user4Actions)
	require.NoError(t, err)
	assert.Len(t, user4Actions, 1, "user 4 should appear exactly once despite being in both followers and watchers")

	// Verify action records were created for each unique audience member.
	var allActions []*hackforger_model.HackforgerAction
	err = db.GetEngine(db.DefaultContext).
		Where("op_type = ? AND act_user_id = 2", hackforger_model.ActionBountyDelivered).
		Find(&allActions)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(allActions), 3, "expected at least actor + 2 targets (followers 4, 8)")
	assert.LessOrEqual(t, len(allActions), 7, "should not exceed actor + all possible unique targets")
}
