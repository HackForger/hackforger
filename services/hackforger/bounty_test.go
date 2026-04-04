// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyForBounty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Bounty 1 is Open, exclusive, publisher=2. Apply as user 5.
	app, err := ApplyForBounty(db.DefaultContext, 1, 5, "I want to work on this")
	require.NoError(t, err)
	assert.Equal(t, int64(1), app.BountyID)
	assert.Equal(t, int64(5), app.UserID)
	assert.Equal(t, hackforger_model.ApplicationStatusPending, app.Status)
	assert.True(t, app.ID > 0, "application should have an ID after creation")
}

func TestApplyForBounty_NotOpen(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Transition bounty 2 to Claimed first so it's no longer Open.
	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 2)
	require.NoError(t, err)
	bounty.Status = hackforger_model.BountyStatusClaimed
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, bounty))

	_, err = ApplyForBounty(db.DefaultContext, 2, 5, "too late")
	require.Error(t, err)
	assert.True(t, IsErrInvalidBountyStatus(err))
}

func TestAcceptApplication_Exclusive(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Application 1: bounty_id=1, user_id=4, pending.
	// Bounty 1: publisher=2, exclusive, Open.
	err := AcceptApplication(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	// Verify bounty is now Claimed with ClaimerID=4.
	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusClaimed, bounty.Status)
	assert.Equal(t, int64(4), bounty.ClaimerID)

	// Verify the application is accepted.
	app, err := hackforger_model.GetBountyApplicationByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.ApplicationStatusAccepted, app.Status)
}

func TestAcceptApplication_NotPublisher(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Application 1: bounty_id=1, publisher=2. Try as user 5.
	err := AcceptApplication(db.DefaultContext, 1, 5)
	require.Error(t, err)
	assert.True(t, IsErrNotPublisher(err))
}

func TestCancelBounty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Bounty 1: publisher=2, Open.
	err := CancelBounty(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusCancelled, bounty.Status)
}

func TestCancelBounty_NotPublisher(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := CancelBounty(db.DefaultContext, 1, 5)
	require.Error(t, err)
	assert.True(t, IsErrNotPublisher(err))
}

func TestCompleteBounty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Set bounty 1 to InReview with ClaimerID=4.
	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	bounty.Status = hackforger_model.BountyStatusInReview
	bounty.ClaimerID = 4
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, bounty))

	err = CompleteBounty(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	bounty, err = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusCompleted, bounty.Status)
}

func TestRejectDelivery(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Set bounty 1 to InReview (Exclusive mode).
	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	bounty.Status = hackforger_model.BountyStatusInReview
	bounty.ClaimerID = 4
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, bounty))

	err = RejectDelivery(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	bounty, err = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusClaimed, bounty.Status)
}

func TestMarkPaid(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Set bounty 1 to Completed.
	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	bounty.Status = hackforger_model.BountyStatusCompleted
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, bounty))

	err = MarkPaid(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	bounty, err = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusPaid, bounty.Status)
}

func TestExclusiveBountyFlow_Happy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Apply as user 5
	_, err := ApplyForBounty(db.DefaultContext, 1, 5, "I can do it")
	require.NoError(t, err)

	// Accept user 5's application -> Claimed
	pending := hackforger_model.ApplicationStatusPending
	apps, _, err := hackforger_model.ListBountyApplications(db.DefaultContext, hackforger_model.ListBountyApplicationsOptions{
		BountyID: 1, UserID: 5, Status: &pending,
	})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.NoError(t, AcceptApplication(db.DefaultContext, apps[0].ID, 2))

	b, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusClaimed, b.Status)
	assert.Equal(t, int64(5), b.ClaimerID)

	// Simulate PR merge -> InReview
	b.Status = hackforger_model.BountyStatusInReview
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, b))

	// Complete -> credits deposited
	require.NoError(t, CompleteBounty(db.DefaultContext, 1, 2))
	b, err = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusCompleted, b.Status)

	// Mark paid
	require.NoError(t, MarkPaid(db.DefaultContext, 1, 2))
	b, err = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusPaid, b.Status)
}

func TestCompetitiveBountyFlow_Happy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Bounty 2 is competitive + open
	_, err := ApplyForBounty(db.DefaultContext, 2, 4, "entry 1")
	require.NoError(t, err)
	_, err = ApplyForBounty(db.DefaultContext, 2, 5, "entry 2")
	require.NoError(t, err)

	// Add a rank-2 reward so both winners can be tested
	require.NoError(t, hackforger_model.CreateBountyReward(db.DefaultContext, &hackforger_model.BountyReward{
		BountyID: 2, Type: hackforger_model.RewardTypeCredits, Credits: 500, Rank: 2, Note: "2nd place",
	}))

	// Start review
	require.NoError(t, StartReview(db.DefaultContext, 2, 2))
	b, err := hackforger_model.GetBountyByID(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusInReview, b.Status)

	// Select winners
	require.NoError(t, SelectWinners(db.DefaultContext, 2, 2, []WinnerInput{
		{UserID: 4, Rank: 1},
		{UserID: 5, Rank: 2},
	}))
	b, err = hackforger_model.GetBountyByID(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusCompleted, b.Status)

	winners, err := hackforger_model.ListBountyWinners(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Len(t, winners, 2)
}

func TestBountyApplication_AcceptRejectsOthers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Two applications for bounty 1 (fixture already has user 4's app)
	_, err := ApplyForBounty(db.DefaultContext, 1, 5, "me")
	require.NoError(t, err)
	_, err = ApplyForBounty(db.DefaultContext, 1, 6, "me too")
	require.NoError(t, err)

	// Accept user 5's application
	pending := hackforger_model.ApplicationStatusPending
	apps, _, err := hackforger_model.ListBountyApplications(db.DefaultContext, hackforger_model.ListBountyApplicationsOptions{
		BountyID: 1, UserID: 5, Status: &pending,
	})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.NoError(t, AcceptApplication(db.DefaultContext, apps[0].ID, 2))

	// User 4's original fixture application + user 6 should be rejected
	rejected := hackforger_model.ApplicationStatusRejected
	_, count, err := hackforger_model.ListBountyApplications(db.DefaultContext, hackforger_model.ListBountyApplicationsOptions{
		BountyID: 1, Status: &rejected,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestCheckExpiredBounties(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Set bounty 1 deadline to past
	b, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	b.Deadline = 1 // Unix epoch + 1 second (far in the past)
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, b))

	require.NoError(t, CheckExpiredBounties(db.DefaultContext))

	b, err = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusExpired, b.Status)
}

func TestCompleteBounty_NoCreditsReward(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create a bounty with only money reward (no credits)
	bounty := &hackforger_model.Bounty{
		RepoID: 1, IssueID: 200, PublisherID: 2, ClaimerID: 4,
		Title: "money only", Status: hackforger_model.BountyStatusInReview,
		Mode: hackforger_model.BountyModeExclusive,
	}
	require.NoError(t, hackforger_model.CreateBounty(db.DefaultContext, bounty))
	require.NoError(t, hackforger_model.CreateBountyReward(db.DefaultContext, &hackforger_model.BountyReward{
		BountyID: bounty.ID, Type: hackforger_model.RewardTypeMoney, Amount: 100, Currency: "USD",
	}))

	// Get initial credit balance
	acct, err := GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	initialBalance := acct.Balance

	require.NoError(t, CompleteBounty(db.DefaultContext, bounty.ID, 2))

	acct, err = GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, initialBalance, acct.Balance, "balance should not change with money-only reward")
}

func TestGetBountyLeaderboard(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Complete a bounty first so leaderboard has data via SelectWinners
	// Use bounty 2 (competitive) for this
	_, err := ApplyForBounty(db.DefaultContext, 2, 4, "entry")
	require.NoError(t, err)
	require.NoError(t, StartReview(db.DefaultContext, 2, 2))
	require.NoError(t, SelectWinners(db.DefaultContext, 2, 2, []WinnerInput{
		{UserID: 4, Rank: 1},
	}))

	stats, err := GetBountyLeaderboard(db.DefaultContext, 10)
	require.NoError(t, err)
	assert.NotEmpty(t, stats)
	assert.Equal(t, int64(4), stats[0].UserID)
	assert.Equal(t, int64(1), stats[0].WinCount)
}
