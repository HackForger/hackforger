// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBounty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	b := &Bounty{
		RepoID: 1, IssueID: 100, PublisherID: 2, Title: "New bounty", Mode: BountyModeExclusive,
	}
	require.NoError(t, CreateBounty(db.DefaultContext, b))
	assert.Greater(t, b.ID, int64(0))
	assert.Equal(t, BountyStatusOpen, b.Status)
}

func TestCreateBounty_DuplicateIssue(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	b := &Bounty{RepoID: 1, IssueID: 1, PublisherID: 2, Title: "duplicate"}
	err := CreateBounty(db.DefaultContext, b)
	assert.True(t, IsErrBountyAlreadyExists(err))
}

func TestGetBountyByIssueID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	b, err := GetBountyByIssueID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), b.ID)
	assert.Equal(t, "Fix authentication bug", b.Title)
}

func TestListBounties_Filter(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	status := BountyStatusOpen
	bounties, count, err := ListBounties(db.DefaultContext, ListBountiesOptions{RepoID: 1, Status: &status})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Len(t, bounties, 2)
}

func TestCreateBountyApplication(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	app := &BountyApplication{BountyID: 1, UserID: 5, Message: "I want to work on this"}
	require.NoError(t, CreateBountyApplication(db.DefaultContext, app))
	assert.Greater(t, app.ID, int64(0))
	assert.Equal(t, ApplicationStatusPending, app.Status)
}

func TestCreateBountyApplication_Duplicate(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	app := &BountyApplication{BountyID: 1, UserID: 4, Message: "duplicate"}
	err := CreateBountyApplication(db.DefaultContext, app)
	assert.True(t, IsErrAlreadyApplied(err))
}

func TestListBountyApplications(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	apps, count, err := ListBountyApplications(db.DefaultContext, ListBountyApplicationsOptions{BountyID: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Len(t, apps, 1)
	assert.Equal(t, int64(4), apps[0].UserID)
}

func TestUpdateBountyApplication(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	app := unittest.AssertExistsAndLoadBean(t, &BountyApplication{ID: 1})
	app.Status = ApplicationStatusAccepted
	require.NoError(t, UpdateBountyApplication(db.DefaultContext, app))
	updated := unittest.AssertExistsAndLoadBean(t, &BountyApplication{ID: 1})
	assert.Equal(t, ApplicationStatusAccepted, updated.Status)
}

func TestCreateBountyReward(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	r := &BountyReward{BountyID: 2, Type: RewardTypeCredits, Credits: 500, Rank: 2}
	require.NoError(t, CreateBountyReward(db.DefaultContext, r))
	assert.Greater(t, r.ID, int64(0))
}

func TestListBountyRewards(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	rewards, err := ListBountyRewards(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Len(t, rewards, 2)
}

func TestDeleteBountyReward(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	require.NoError(t, DeleteBountyReward(db.DefaultContext, 1))
	rewards, err := ListBountyRewards(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Len(t, rewards, 1)
}

func TestCreateBountyWinner(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	w := &BountyWinner{BountyID: 2, UserID: 4, Rank: 1}
	require.NoError(t, CreateBountyWinner(db.DefaultContext, w))
	assert.Greater(t, w.ID, int64(0))
}

func TestListBountyWinners(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	w := &BountyWinner{BountyID: 2, UserID: 4, Rank: 1}
	require.NoError(t, CreateBountyWinner(db.DefaultContext, w))
	winners, err := ListBountyWinners(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Len(t, winners, 1)
	assert.Equal(t, int64(4), winners[0].UserID)
}
