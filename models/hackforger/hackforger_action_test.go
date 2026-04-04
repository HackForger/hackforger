// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHackforgerFeeds_Global(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	action := &hackforger_model.HackforgerAction{
		UserID:     0,
		ActUserID:  2,
		OpType:     30,
		EntityType: "hackathon",
		EntityID:   1,
		EntityName: "Test Hackathon",
	}
	_, err := db.GetEngine(db.DefaultContext).Insert(action)
	require.NoError(t, err)

	feeds, count, err := hackforger_model.GetHackforgerFeeds(db.DefaultContext, hackforger_model.GetHackforgerFeedsOptions{
		UserID:        99,
		IncludeGlobal: true,
		ListOptions:   db.ListOptions{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.True(t, count >= 1)
	assert.True(t, len(feeds) >= 1)
}

func TestGetHackforgerFeeds_ByActUser(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	action := &hackforger_model.HackforgerAction{
		UserID:     5,
		ActUserID:  2,
		OpType:     34,
		EntityType: "bounty",
		EntityID:   10,
		EntityName: "Fix login bug",
	}
	_, err := db.GetEngine(db.DefaultContext).Insert(action)
	require.NoError(t, err)

	feeds, count, err := hackforger_model.GetHackforgerFeeds(db.DefaultContext, hackforger_model.GetHackforgerFeedsOptions{
		ActUserID:   2,
		ListOptions: db.ListOptions{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.True(t, count >= 1)
	found := false
	for _, f := range feeds {
		if f.EntityName == "Fix login bug" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestLoadActUsers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	actions := []*hackforger_model.HackforgerAction{
		{ActUserID: 1},
		{ActUserID: 2},
		{ActUserID: 1},
	}
	err := hackforger_model.LoadActUsers(db.DefaultContext, actions)
	require.NoError(t, err)
	assert.NotNil(t, actions[0].ActUser)
	assert.NotNil(t, actions[1].ActUser)
	assert.Equal(t, actions[0].ActUser, actions[2].ActUser)
}
