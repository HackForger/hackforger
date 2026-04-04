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

func TestGetGrantProjectByUserAndRound(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Existing project: user 4 in round 2
	p, err := hackforger_model.GetGrantProjectByUserAndRound(db.DefaultContext, 4, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), p.ID)
	assert.Equal(t, "Eve's Project", p.Title)

	// Non-existent combination
	_, err = hackforger_model.GetGrantProjectByUserAndRound(db.DefaultContext, 99, 99)
	assert.True(t, hackforger_model.IsErrGrantProjectNotExist(err))
}

func TestDeleteGrantProject_OnlyPending(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Project 1 is Pending (status=0), should delete successfully
	err := hackforger_model.DeleteGrantProject(db.DefaultContext, 1)
	require.NoError(t, err)
	_, err = hackforger_model.GetGrantProjectByID(db.DefaultContext, 1)
	assert.True(t, hackforger_model.IsErrGrantProjectNotExist(err))

	// Project 2 is Approved (status=1), should fail
	err = hackforger_model.DeleteGrantProject(db.DefaultContext, 2)
	assert.Error(t, err)

	// Non-existent project
	err = hackforger_model.DeleteGrantProject(db.DefaultContext, 999)
	assert.True(t, hackforger_model.IsErrGrantProjectNotExist(err))
}

func TestCountGrantProjectsByRound(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Round 3 has 2 projects (both Approved)
	count, err := hackforger_model.CountGrantProjectsByRound(db.DefaultContext, 3, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Round 3, filter by Approved
	approved := hackforger_model.GrantProjectStatusApproved
	count, err = hackforger_model.CountGrantProjectsByRound(db.DefaultContext, 3, &approved)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Round 3, filter by Pending (should be 0)
	pending := hackforger_model.GrantProjectStatusPending
	count, err = hackforger_model.CountGrantProjectsByRound(db.DefaultContext, 3, &pending)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Round 2 has 1 project
	count, err = hackforger_model.CountGrantProjectsByRound(db.DefaultContext, 2, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestListGrantProjectsByRound_FilterByStatus(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// List all projects in round 3
	projects, count, err := hackforger_model.ListGrantProjectsByRound(db.DefaultContext, hackforger_model.ListGrantProjectsByRoundOptions{
		RoundID: 3,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Len(t, projects, 2)

	// Filter by Approved status in round 3
	approved := hackforger_model.GrantProjectStatusApproved
	projects, count, err = hackforger_model.ListGrantProjectsByRound(db.DefaultContext, hackforger_model.ListGrantProjectsByRoundOptions{
		RoundID: 3,
		Status:  &approved,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	for _, p := range projects {
		assert.Equal(t, hackforger_model.GrantProjectStatusApproved, p.Status)
	}

	// Filter by Pending status in round 3 (should be empty)
	pending := hackforger_model.GrantProjectStatusPending
	projects, count, err = hackforger_model.ListGrantProjectsByRound(db.DefaultContext, hackforger_model.ListGrantProjectsByRoundOptions{
		RoundID: 3,
		Status:  &pending,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
	assert.Empty(t, projects)
}
